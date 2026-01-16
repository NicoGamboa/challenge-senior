package wallet

func ToCreditRequest(userID string, amount int64) CreditRequest {
	return CreditRequest{UserID: userID, Amount: amount}
}
