// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// KubeOneDigitalOceanCloudSpec KubeOneDigitalOceanCloudSpec specifies access data to DigitalOcean.
//
// swagger:model KubeOneDigitalOceanCloudSpec
type KubeOneDigitalOceanCloudSpec struct {

	// Token is used to authenticate with the DigitalOcean API.
	Token string `json:"token,omitempty"`
}

// Validate validates this kube one digital ocean cloud spec
func (m *KubeOneDigitalOceanCloudSpec) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this kube one digital ocean cloud spec based on context it is used
func (m *KubeOneDigitalOceanCloudSpec) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *KubeOneDigitalOceanCloudSpec) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *KubeOneDigitalOceanCloudSpec) UnmarshalBinary(b []byte) error {
	var res KubeOneDigitalOceanCloudSpec
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}