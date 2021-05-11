// Code generated by go-swagger; DO NOT EDIT.

package seed

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

// GetSeedSettingsReader is a Reader for the GetSeedSettings structure.
type GetSeedSettingsReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *GetSeedSettingsReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewGetSeedSettingsOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewGetSeedSettingsUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewGetSeedSettingsForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewGetSeedSettingsDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewGetSeedSettingsOK creates a GetSeedSettingsOK with default headers values
func NewGetSeedSettingsOK() *GetSeedSettingsOK {
	return &GetSeedSettingsOK{}
}

/*GetSeedSettingsOK handles this case with default header values.

SeedSettings
*/
type GetSeedSettingsOK struct {
	Payload *models.SeedSettings
}

func (o *GetSeedSettingsOK) Error() string {
	return fmt.Sprintf("[GET /api/v2/seeds/{seed_name}/settings][%d] getSeedSettingsOK  %+v", 200, o.Payload)
}

func (o *GetSeedSettingsOK) GetPayload() *models.SeedSettings {
	return o.Payload
}

func (o *GetSeedSettingsOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.SeedSettings)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewGetSeedSettingsUnauthorized creates a GetSeedSettingsUnauthorized with default headers values
func NewGetSeedSettingsUnauthorized() *GetSeedSettingsUnauthorized {
	return &GetSeedSettingsUnauthorized{}
}

/*GetSeedSettingsUnauthorized handles this case with default header values.

EmptyResponse is a empty response
*/
type GetSeedSettingsUnauthorized struct {
}

func (o *GetSeedSettingsUnauthorized) Error() string {
	return fmt.Sprintf("[GET /api/v2/seeds/{seed_name}/settings][%d] getSeedSettingsUnauthorized ", 401)
}

func (o *GetSeedSettingsUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewGetSeedSettingsForbidden creates a GetSeedSettingsForbidden with default headers values
func NewGetSeedSettingsForbidden() *GetSeedSettingsForbidden {
	return &GetSeedSettingsForbidden{}
}

/*GetSeedSettingsForbidden handles this case with default header values.

EmptyResponse is a empty response
*/
type GetSeedSettingsForbidden struct {
}

func (o *GetSeedSettingsForbidden) Error() string {
	return fmt.Sprintf("[GET /api/v2/seeds/{seed_name}/settings][%d] getSeedSettingsForbidden ", 403)
}

func (o *GetSeedSettingsForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewGetSeedSettingsDefault creates a GetSeedSettingsDefault with default headers values
func NewGetSeedSettingsDefault(code int) *GetSeedSettingsDefault {
	return &GetSeedSettingsDefault{
		_statusCode: code,
	}
}

/*GetSeedSettingsDefault handles this case with default header values.

errorResponse
*/
type GetSeedSettingsDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the get seed settings default response
func (o *GetSeedSettingsDefault) Code() int {
	return o._statusCode
}

func (o *GetSeedSettingsDefault) Error() string {
	return fmt.Sprintf("[GET /api/v2/seeds/{seed_name}/settings][%d] getSeedSettings default  %+v", o._statusCode, o.Payload)
}

func (o *GetSeedSettingsDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *GetSeedSettingsDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
