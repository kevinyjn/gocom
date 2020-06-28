package soapclient

type wsdlObject struct {
	Services []wsdlService     `xml:"service"`
	Bindings []wsdlPortBinding `xml:"binding"`
}

type wsdlService struct {
	Name  string     `xml:"name,attr"`
	Ports []wsdlPort `xml:"port"`
}

type wsdlPort struct {
	Name    string          `xml:"name,attr"`
	Address wsdlPortAddress `xml:"address"`
	Binding string          `xml:"binding,attr"`
}

type wsdlPortAddress struct {
	Location string `xml:"location,attr"`
}

type wsdlPortBinding struct {
	Name       string                 `xml:"name,attr"`
	Type       string                 `xml:"type,attr"`
	Operations []wsdlBindingOperation `xml:"operation"`
}

type wsdlBindingOperation struct {
	Name      string                     `xml:"name,attr"`
	Operation wsdlBindingOperationAction `xml:"operation"`
}

type wsdlBindingOperationAction struct {
	SoapAction string `xml:"soapAction,attr"`
}
