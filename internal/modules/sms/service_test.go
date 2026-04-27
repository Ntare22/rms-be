package sms

import "testing"

func TestRenderTemplate(t *testing.T) {
	out, err := RenderTemplate("Hi {{name}}, ref {{ref}}", map[string]string{
		"name": "Jane",
		"ref":  "INV-9",
	})
	if err != nil {
		t.Fatalf("RenderTemplate err: %v", err)
	}
	if out != "Hi Jane, ref INV-9" {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestRenderTemplate_Unresolved(t *testing.T) {
	_, err := RenderTemplate("Hi {{name}} {{missing}}", map[string]string{"name": "Jane"})
	if err == nil {
		t.Fatalf("expected unresolved error")
	}
}

func TestNormalizePhone(t *testing.T) {
	got, err := NormalizePhone("0712 345 678")
	if err != nil {
		t.Fatalf("NormalizePhone err: %v", err)
	}
	if got != "+254712345678" {
		t.Fatalf("expected +254..., got %s", got)
	}
}

func TestNormalizePhone_Invalid(t *testing.T) {
	if _, err := NormalizePhone("abc"); err == nil {
		t.Fatalf("expected invalid phone error")
	}
}
