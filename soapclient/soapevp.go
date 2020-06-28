package soapclient

import (
	"encoding/xml"
)

// SOAPEncoder interface
type SOAPEncoder interface {
	Encode(v interface{}) error
	Flush() error
}

// SOAPDecoder interface
type SOAPDecoder interface {
	Decode(v interface{}) error
}

// SOAPEnvelope struct
type SOAPEnvelope struct {
	XMLName xml.Name      `xml:"soap-env:Envelope"`
	NS      string        `xml:"xmlns:soap-env,attr"`
	Headers []interface{} `xml:"http://schemas.xmlsoap.org/soap/envelope/ Header,omitempty"`
	Body    SOAPBody
}

// SOAPBody struct
type SOAPBody struct {
	XMLName xml.Name `xml:"soap-env:Body"`
	// NS      string   `xml:"xmlns:soap-env,attr"`

	Fault   *SOAPFault  `xml:",omitempty"`
	Content interface{} `xml:",omitempty"`
}

// SOAPEnvelopeResponse struct
type SOAPEnvelopeResponse struct {
	XMLName xml.Name      `xml:"http://schemas.xmlsoap.org/soap/envelope/ Envelope"`
	Headers []interface{} `xml:"http://schemas.xmlsoap.org/soap/envelope/ Header,omitempty"`
	Body    SOAPBodyResponse
}

// SOAPBodyResponse struct
type SOAPBodyResponse struct {
	XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Body"`

	Fault   *SOAPFault  `xml:",omitempty"`
	Content interface{} `xml:",omitempty"`
}

// NewSOAPEnvelope envelope
func NewSOAPEnvelope(content interface{}, headers []interface{}) *SOAPEnvelope {
	return &SOAPEnvelope{
		NS:      "http://schemas.xmlsoap.org/soap/envelope/",
		Headers: headers,
		Body: SOAPBody{
			Content: content,
		},
	}
}

// UnmarshalXML unmarshals SOAPBody xml
func (b *SOAPBodyResponse) UnmarshalXML(d *xml.Decoder, _ xml.StartElement) error {
	if b.Content == nil {
		return xml.UnmarshalError("Content must be a pointer to a struct")
	}

	var (
		token    xml.Token
		err      error
		consumed bool
	)

Loop:
	for {
		if token, err = d.Token(); err != nil {
			return err
		}

		if token == nil {
			break
		}

		switch se := token.(type) {
		case xml.StartElement:
			if consumed {
				return xml.UnmarshalError("Found multiple elements inside SOAP body; not wrapped-document/literal WS-I compliant")
			} else if se.Name.Space == "http://schemas.xmlsoap.org/soap/envelope/" && se.Name.Local == "Fault" {
				b.Fault = &SOAPFault{}
				b.Content = nil

				err = d.DecodeElement(b.Fault, &se)
				if err != nil {
					return err
				}

				consumed = true
			} else {
				if err = d.DecodeElement(b.Content, &se); err != nil {
					return err
				}

				consumed = true
			}
		case xml.EndElement:
			break Loop
		}
	}

	return nil
}

// SOAPFault struct
type SOAPFault struct {
	XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Fault"`

	Code   string `xml:"faultcode,omitempty"`
	String string `xml:"faultstring,omitempty"`
	Actor  string `xml:"faultactor,omitempty"`
	Detail string `xml:"detail,omitempty"`
}

// Error get error
func (f *SOAPFault) Error() string {
	return f.String
}

// Constants
const (
	// Predefined WSS namespaces to be used in
	WssNsWSSE       string = "http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd"
	WssNsWSU        string = "http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd"
	WssNsType       string = "http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-username-token-profile-1.0#PasswordText"
	mtomContentType string = `multipart/related; start-info="application/soap+xml"; type="application/xop+xml"; boundary="%s"`
)

// WSSSecurityHeader struct
type WSSSecurityHeader struct {
	XMLName   xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ wsse:Security"`
	XMLNSWsse string   `xml:"xmlns:wsse,attr"`

	MustUnderstand string `xml:"mustUnderstand,attr,omitempty"`

	Token *WSSUsernameToken `xml:",omitempty"`
}

// WSSUsernameToken struct
type WSSUsernameToken struct {
	XMLName   xml.Name `xml:"wsse:UsernameToken"`
	XMLNSWsu  string   `xml:"xmlns:wsu,attr"`
	XMLNSWsse string   `xml:"xmlns:wsse,attr"`

	ID string `xml:"wsu:Id,attr,omitempty"`

	Username *WSSUsername `xml:",omitempty"`
	Password *WSSPassword `xml:",omitempty"`
}

// WSSUsername struct
type WSSUsername struct {
	XMLName   xml.Name `xml:"wsse:Username"`
	XMLNSWsse string   `xml:"xmlns:wsse,attr"`

	Data string `xml:",chardata"`
}

// WSSPassword struct
type WSSPassword struct {
	XMLName   xml.Name `xml:"wsse:Password"`
	XMLNSWsse string   `xml:"xmlns:wsse,attr"`
	XMLNSType string   `xml:"Type,attr"`

	Data string `xml:",chardata"`
}

// NewWSSSecurityHeader creates WSSSecurityHeader instance
func NewWSSSecurityHeader(user, pass, tokenID, mustUnderstand string) *WSSSecurityHeader {
	hdr := &WSSSecurityHeader{XMLNSWsse: WssNsWSSE, MustUnderstand: mustUnderstand}
	hdr.Token = &WSSUsernameToken{XMLNSWsu: WssNsWSU, XMLNSWsse: WssNsWSSE, ID: tokenID}
	hdr.Token.Username = &WSSUsername{XMLNSWsse: WssNsWSSE, Data: user}
	hdr.Token.Password = &WSSPassword{XMLNSWsse: WssNsWSSE, XMLNSType: WssNsType, Data: pass}
	return hdr
}
