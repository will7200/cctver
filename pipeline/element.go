package pipeline

import (
	"errors"
	"fmt"

	"github.com/go-gst/go-glib/glib"
	"github.com/go-gst/go-gst/gst"
)

// Element struct wraps a gst element
type Element struct {
	Factory    string
	Name       string
	Properties map[string]interface{}

	el *gst.Element
}

func (e *Element) Build() error {
	if e.el != nil {
		return nil
	}
	element, err := gst.NewElementWithName(e.Factory, e.Name)
	if err != nil {
		return err
	}
	if e.Properties != nil && len(e.Properties) > 0 {
		for k, v := range e.Properties {
			t, err := element.GetPropertyType(k)
			if err != nil {
				return fmt.Errorf("unable got get %s: %w", k, err)
			}
			var value *glib.Value
			switch true {
			case t.IsA(glib.TYPE_ENUM) == true:
				value, err = glib.ValueInit(t)
				if err != nil {
					return err
				}
				if _, ok := v.(int); !ok {
					return fmt.Errorf("property %s expecting an int", k)
				}
				value.SetEnum(v.(int))
				if err = element.SetPropertyValue(k, value); err != nil {
					return err
				}
			case t.IsA(glib.TYPE_FLAGS) == true:
				value, err = glib.ValueInit(t)
				if err != nil {
					return err
				}
				if _, ok := v.(uint); !ok {
					return fmt.Errorf("property %s expecting an uint", k)
				}
				value.SetFlags(v.(uint))
				if err = element.SetPropertyValue(k, value); err != nil {
					return err
				}
			default:
				if err = element.Set(k, v); err != nil {
					return err
				}
			}
		}
	}
	e.el = element
	return nil
}

func (e *Element) Link(other *Element) error {
	if e.el == nil || other.el == nil {
		return errors.New("unable to link un-built elements")
	}
	return e.el.Link(other.el)
}

func (e *Element) LinkFiltered(other *Element, filter *gst.Caps) error {
	if e.el == nil || other.el == nil {
		return errors.New("unable to link un-built elements")
	}
	return e.el.LinkFiltered(other.el, filter)
}

func (e *Element) Unlink(other *Element) error {
	if e.el == nil || other.el == nil {
		return errors.New("unable to link un-built elements")
	}
	e.el.Unlink(other.el)
	return nil
}

func NewFileSrcElement(name string, file string) *Element {
	return &Element{
		Factory: "filesrc",
		Name:    name,
		Properties: map[string]interface{}{
			"location": file,
		},
		el: nil,
	}
}

func NewDecodeElement(name string) *Element {
	return &Element{
		Factory:    "decodebin",
		Name:       name,
		Properties: nil,
		el:         nil,
	}
}
