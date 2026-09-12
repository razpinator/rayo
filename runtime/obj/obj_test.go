package obj

import "testing"

func TestObj(t *testing.T) {
	o := NewObj()
	o.SetAttr("foo", 123)
	if o.GetAttr("foo") != 123 {
		t.Errorf("GetAttr failed")
	}
	o.SetAttr("bar", "baz")
	if o.GetAttr("bar") != "baz" {
		t.Errorf("SetAttr failed")
	}
}

func TestHasAttrAndNilSafe(t *testing.T) {
	o := NewObj()
	o.SetAttr("x", 1)
	if !o.HasAttr("x") || o.HasAttr("y") {
		t.Errorf("HasAttr failed")
	}
	var nilObj *Obj
	if nilObj.GetAttr("x") != nil {
		t.Errorf("GetAttr on nil should be nil")
	}
	if nilObj.HasAttr("x") {
		t.Errorf("HasAttr on nil should be false")
	}
}

func TestSafeGet(t *testing.T) {
	if SafeGet(nil, "a") != nil {
		t.Errorf("SafeGet(nil) should be nil")
	}
	o := NewObj()
	o.SetAttr("a", 5)
	if SafeGet(o, "a") != 5 {
		t.Errorf("SafeGet on *Obj failed")
	}
	m := map[string]any{"a": 6}
	if SafeGet(m, "a") != 6 {
		t.Errorf("SafeGet on map failed")
	}
}

func TestGetPath(t *testing.T) {
	profile := NewObj()
	profile.SetAttr("email", "alice@example.com")
	user := NewObj()
	user.SetAttr("profile", profile)

	if v, ok := GetPath(user, "profile.email"); !ok || v != "alice@example.com" {
		t.Errorf("GetPath resolved wrong value: %v %v", v, ok)
	}
	// Missing middle segment (safe navigation short-circuits).
	empty := NewObj()
	if _, ok := GetPath(empty, "profile.email"); ok {
		t.Errorf("GetPath should fail on missing segment")
	}
	// Mixed map/obj traversal.
	mixed := map[string]any{"profile": map[string]any{"email": "bob@example.com"}}
	if v, _ := GetPath(mixed, "profile.email"); v != "bob@example.com" {
		t.Errorf("GetPath over maps failed: %v", v)
	}
	// Default fallback.
	if got := GetPathOr(empty, "profile.email", "no email"); got != "no email" {
		t.Errorf("GetPathOr default failed: %v", got)
	}
}
