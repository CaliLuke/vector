//go:build integration

package mutationreceipts_test

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	receipts "github.com/CaliLuke/go-plane/internal/mutationreceipts"
	testsupport "github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// dropCommitAcknowledgement relays the real PostgreSQL wire protocol, then closes
// the connection after the server sends CommandComplete(COMMIT), before delivering
// that response to pgx. This exercises a committed write with an actual lost ack.
func dropCommitAcknowledgement(t *testing.T, connection string) (*pgxpool.Pool, <-chan struct{}) {
	t.Helper()
	config, err := pgxpool.ParseConfig(connection)
	require.NoError(t, err)
	destination := net.JoinHostPort(config.ConnConfig.Host, strconv.Itoa(int(config.ConnConfig.Port)))
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	closeListener := sync.OnceFunc(func() { testsupport.Close(t, listener) })
	dropped, finished := make(chan struct{}), make(chan struct{})
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(func() {
		cancel()
		closeListener()
		select {
		case <-finished:
		case <-time.After(5 * time.Second):
			t.Error("PostgreSQL proxy did not stop")
		}
	})
	go func() {
		defer close(finished)
		client, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		closeListener() // A pool replacement must fail promptly, not wait for a second accept.
		closeClient := sync.OnceFunc(func() { testsupport.Close(t, client) })
		defer closeClient()
		server, dialErr := (&net.Dialer{}).DialContext(ctx, "tcp", destination)
		if dialErr != nil {
			return
		}
		closeConnections := sync.OnceFunc(func() {
			closeClient()
			testsupport.Close(t, server)
		})
		defer closeConnections()
		stop := context.AfterFunc(ctx, closeConnections)
		defer stop()
		// StartupMessage has a length prefix but no message type.
		startup, readErr := readPGPacket(client)
		if readErr != nil {
			return
		}
		if _, writeErr := server.Write(startup); writeErr != nil {
			return
		}
		var committing atomic.Bool
		forwarding := make(chan struct{})
		go func() {
			defer close(forwarding)
			for {
				kind := make([]byte, 1)
				if _, readErr := io.ReadFull(client, kind); readErr != nil {
					return
				}
				packet, readErr := readPGPacket(client)
				if readErr != nil {
					return
				}
				if kind[0] == 'Q' && strings.EqualFold(strings.TrimSuffix(string(packet[4:]), "\x00"), "commit") {
					committing.Store(true)
				}
				if _, writeErr := server.Write(append(kind, packet...)); writeErr != nil {
					return
				}
			}
		}()
		defer func() { closeConnections(); <-forwarding }()
		for {
			kind := make([]byte, 1)
			if _, readErr := io.ReadFull(server, kind); readErr != nil {
				return
			}
			packet, readErr := readPGPacket(server)
			if readErr != nil {
				return
			}
			if committing.Load() && kind[0] == 'C' && string(packet[4:]) == "COMMIT\x00" {
				close(dropped)
				return
			}
			if _, writeErr := client.Write(append(kind, packet...)); writeErr != nil {
				return
			}
		}
	}()
	address := listener.Addr().(*net.TCPAddr)
	config.ConnConfig.Host = "127.0.0.1"
	config.ConnConfig.Port = uint16(address.Port)
	config.ConnConfig.TLSConfig = nil
	config.ConnConfig.Fallbacks = nil
	config.MaxConns = 1
	pool, err := pgxpool.NewWithConfig(t.Context(), config)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	return pool, dropped
}

func readPGPacket(reader io.Reader) ([]byte, error) {
	header := make([]byte, 4)
	if _, err := io.ReadFull(reader, header); err != nil {
		return nil, err
	}
	length := int(binary.BigEndian.Uint32(header))
	if length < 4 || length > 4<<20 {
		return nil, errors.New("invalid PostgreSQL fixture packet")
	}
	packet := make([]byte, length)
	copy(packet, header)
	_, err := io.ReadFull(reader, packet[4:])
	return packet, err
}

func TestLostCommitAcknowledgementIsIndeterminateAndKeyedRetryReplays(t *testing.T) {
	for _, keyed := range []bool{true, false} {
		t.Run(map[bool]string{true: "keyed", false: "unkeyed"}[keyed], func(t *testing.T) {
			f := receiptDatabase(t)
			if !keyed {
				f.command.RequestID = nil
			}
			proxy, dropped := dropCommitAcknowledgement(t, f.url)
			ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
			defer cancel()
			result, err := receipts.Execute(ctx, proxy, f.command, operation())
			signal(t, dropped)
			require.ErrorIs(t, err, receipts.ErrIndeterminate)
			require.Empty(t, result)
			require.NotNil(t, errors.Unwrap(err), "internal commit cause must remain inspectable")
			require.Equal(t, receipts.ErrIndeterminate.Error(), err.Error(), "public error must not contain database details")
			if keyed {
				assertCounts(t, f, 1, 1, 1)
				replay, replayErr := receipts.Execute(t.Context(), f.db, f.command, operation())
				require.NoError(t, replayErr)
				require.True(t, replay.Replayed)
				require.JSONEq(t, `{"id":1,"revision":1}`, string(replay.Value))
				assertCounts(t, f, 1, 1, 1)
			} else {
				assertCounts(t, f, 1, 1, 0)
			}
		})
	}
}
