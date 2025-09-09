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
