// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// Kubevirt kubevirt
//
// swagger:model Kubevirt
type Kubevirt struct {

	// datacenter
	Datacenter string `json:"datacenter,omitempty"`

	// enabled
	Enabled bool `json:"enabled,omitempty"`

	// kubeconfig
	Kubeconfig string `json:"kubeconfig,omitempty"`
}

// Validate validates this kubevirt
func (m *Kubevirt) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this kubevirt based on context it is used
func (m *Kubevirt) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *Kubevirt) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *Kubevirt) UnmarshalBinary(b []byte) error {
	var res Kubevirt
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}