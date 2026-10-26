export function waitForDom(container, predicate, message, timeoutMs = 2000) {
  return new Promise((resolve, reject) => {
    let timeout;
    const observer = new MutationObserver(() => settleIfReady());
    const settle = (error) => {
      observer.disconnect();
      clearTimeout(timeout);
      if (error) reject(error);
      else resolve();
    };
    const settleIfReady = () => {
      if (predicate()) settle();
    };

    timeout = setTimeout(() => settle(new Error(message)), timeoutMs);
    observer.observe(container, { attributes: true, childList: true, characterData: true, subtree: true });
    settleIfReady();
  });
}
