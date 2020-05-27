package bzz

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"

	"github.com/ethersphere/bee/pkg/crypto"
	"github.com/ethersphere/bee/pkg/swarm"

	ma "github.com/multiformats/go-multiaddr"
)

var ErrInvalidAddress = errors.New("invalid address")

type Address struct {
	Underlay  ma.Multiaddr
	Overlay   swarm.Address
	Signature []byte
}

func NewAddress(signer crypto.Signer, underlay ma.Multiaddr, overlay swarm.Address, networkID uint64) (*Address, error) {
	networkIDBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(networkIDBytes, networkID)
	signature, err := signer.Sign(append(underlay.Bytes(), networkIDBytes...))
	if err != nil {
		return nil, err
	}

	return &Address{
		Underlay:  underlay,
		Overlay:   overlay,
		Signature: signature,
	}, nil
}

func Parse(underlay, overlay, signature []byte, networkID uint64) (*Address, error) {
	networkIDBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(networkIDBytes, networkID)
	recoveredPK, err := crypto.Recover(signature, append(underlay, networkIDBytes...))
	if err != nil {
		return nil, ErrInvalidAddress
	}

	recoveredOverlay := crypto.NewOverlayAddress(*recoveredPK, networkID)
	if !bytes.Equal(recoveredOverlay.Bytes(), overlay) {
		return nil, ErrInvalidAddress
	}

	multiUnderlay, err := ma.NewMultiaddrBytes(underlay)
	if err != nil {
		return nil, ErrInvalidAddress
	}

	return &Address{
		Underlay:  multiUnderlay,
		Overlay:   swarm.NewAddress(overlay),
		Signature: signature,
	}, nil
}

func (a *Address) Equal(b *Address) bool {
	return a.Overlay.Equal(b.Overlay) && a.Underlay.Equal(b.Underlay) && bytes.Equal(a.Signature, b.Signature)
}

func (p *Address) MarshalJSON() ([]byte, error) {
	v := struct {
		Overlay   string
		Underlay  string
		Signature string
	}{
		Overlay:   p.Overlay.String(),
		Underlay:  p.Underlay.String(),
		Signature: base64.StdEncoding.EncodeToString(p.Signature),
	}
	return json.Marshal(&v)
}

func (p *Address) UnmarshalJSON(b []byte) error {
	v := struct {
		Overlay   string
		Underlay  string
		Signature string
	}{}
	err := json.Unmarshal(b, &v)
	if err != nil {
		return err
	}

	a, err := swarm.ParseHexAddress(v.Overlay)
	if err != nil {
		return err
	}

	p.Overlay = a

	m, err := ma.NewMultiaddr(v.Underlay)
	if err != nil {
		return err
	}

	p.Underlay = m
	p.Signature, err = base64.StdEncoding.DecodeString(v.Signature)
	if err != nil {
		return err
	}

	return nil
}