package shared

type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

func ParseRole(s string) Role {
	switch s {
	case "admin":
		return RoleAdmin
	default:
		return RoleUser
	}
}
