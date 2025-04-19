package encxerr

type Action int8

const (
	Unknown Action = iota
	BasicHash
	SecureHash
	Encrypt
	Decrypt
)

var actions = [3]string{
	"encrypt",
	"basic hash",
	"secure hash",
}

func (a Action) String() string {
	actions := map[Action]string{
		Unknown:    "unknown",
		BasicHash:  "basic hash",
		SecureHash: "secure hash",
		Encrypt:    "encrypt",
		Decrypt:    "decrypt",
	}

	if str, ok := actions[a]; ok {
		return str
	}
	return "unknown"
}
