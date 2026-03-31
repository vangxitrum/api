package models

type ResponseSuccess struct {
	Status  string `json:"status"`
	Message string `json:"message"`
} //	@name	ResponseSuccess

// swagger:model
type ResponseError struct {
	Status  string `json:"status"`
	Message string `json:"message"`
} //	@name	ResponseError
