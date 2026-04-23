package server

import _ "embed"

// The PEM files are copied from ../../certs/ so //go:embed can read them
// (embed does not follow symlinks). Refresh with: go generate ./internal/server

//go:generate sh -c "cp ../../certs/fullchain.pem embed/fullchain.pem"
//go:embed embed/fullchain.pem
var CertPEM []byte

//go:generate sh -c "cp ../../certs/key.pem embed/key.pem"
//go:embed embed/key.pem
var KeyPEM []byte
