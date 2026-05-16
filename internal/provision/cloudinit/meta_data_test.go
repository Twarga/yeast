package cloudinit

import "testing"

func TestRenderMetaDataStable(t *testing.T) {
	t.Parallel()

	first, err := RenderMetaData(MetaDataInput{Hostname: "web"})
	if err != nil {
		t.Fatalf("RenderMetaData returned error: %v", err)
	}
	second, err := RenderMetaData(MetaDataInput{Hostname: "web"})
	if err != nil {
		t.Fatalf("RenderMetaData returned error on second render: %v", err)
	}

	want := "instance-id: web\nlocal-hostname: web\n"
	if first != want {
		t.Fatalf("unexpected meta-data:\n got: %q\nwant: %q", first, want)
	}
	if second != want {
		t.Fatalf("unexpected second meta-data:\n got: %q\nwant: %q", second, want)
	}
}
