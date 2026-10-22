These fixtures and replay logs are preserved as historical evidence from before
the external replay harness cleanup fix. The target replay previously sent
unseeded placeholder session cookies and used Django's generic session model
for cleanup instead of Plane's session model. The 16 source fixtures were
recaptured with the corrected setup and are byte-identical to this archive; the
historical replay results are not current Go parity evidence.
