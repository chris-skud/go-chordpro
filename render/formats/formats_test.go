package formats

import "testing"

func TestLookup(t *testing.T) {
	for name, ext := range map[string]string{"text": "txt", "txt": "txt", "html": "html", "pdf": "pdf"} {
		f, err := Lookup(name)
		if err != nil {
			t.Fatalf("Lookup(%q): %v", name, err)
		}
		if f.Ext != ext || f.ContentType == "" || f.New == nil {
			t.Errorf("Lookup(%q) = %+v", name, f)
		}
	}
	if _, err := Lookup("docx"); err == nil {
		t.Error("Lookup(docx): want error")
	}
}
