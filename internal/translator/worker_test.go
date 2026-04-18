package translator

import "testing"

func TestCompactWorkerErrorMessage(t *testing.T) {
	t.Parallel()

	raw := "Translation subprocess initialization error: Your request was blocked.: settings.basic.input_files is for cli & config, pdf2zh_next.highlevel.do_translate_async_stream will ignore this field and only translate the file pointed to by the file parameter.\r\nReceived error from subprocess: Translation subprocess initialization error: Your request was blocked.\r\nTraceback: Traceback (most recent call last):"
	got := compactWorkerErrorMessage(raw)
	want := "Translation subprocess initialization error: Your request was blocked."
	if got != want {
		t.Fatalf("compactWorkerErrorMessage() = %q, want %q", got, want)
	}
}
