package user

type User struct {
	ID         int
	Email      string
	Name       string
	Provider   string
	ProviderID string
	RefreshToken string
}

