package domain

type User struct {
	ID           int64
	Login        string
	PasswordHash string
}

type Credentials struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}
