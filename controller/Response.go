// controller/Response.go
package controller

// ButtonType tipo de botón
type ButtonType int

const (
	ButtonInline ButtonType = iota // Botón inline (debajo del mensaje)
	ButtonReply                    // Botón reply (reemplaza teclado)
)

// Button representa un botón
type Button struct {
	Text string
	Data string     // callback_data para inline, o texto a enviar para reply
	Type ButtonType // inline o reply
}

// Response respuesta estructurada del bot
type Response struct {
	Text       string
	Buttons    []Button // lista plana de botones
	ForceReply bool     // true si queremos forzar respuesta del usuario
}

// NewResponse crea una respuesta solo con texto
func NewResponse(text string) *Response {
	return &Response{
		Text:    text,
		Buttons: []Button{},
	}
}

// WithButtons añade botones a la respuesta
func (r *Response) WithButtons(buttons ...Button) *Response {
	r.Buttons = append(r.Buttons, buttons...)
	return r
}

// HasButtons retorna true si hay botones
func (r *Response) HasButtons() bool {
	return len(r.Buttons) > 0
}

// HasInlineButtons retorna true si hay botones inline
func (r *Response) HasInlineButtons() bool {
	for _, btn := range r.Buttons {
		if btn.Type == ButtonInline {
			return true
		}
	}
	return false
}

// HasReplyButtons retorna true si hay botones reply
func (r *Response) HasReplyButtons() bool {
	for _, btn := range r.Buttons {
		if btn.Type == ButtonReply {
			return true
		}
	}
	return false
}
