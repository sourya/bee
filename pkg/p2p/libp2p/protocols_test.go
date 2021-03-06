// Copyright 2020 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package libp2p_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ethersphere/bee/pkg/p2p"
	"github.com/ethersphere/bee/pkg/p2p/libp2p"
	"github.com/multiformats/go-multistream"
)

func TestNewStream(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s1, overlay1 := newService(t, 1, libp2p.Options{})

	s2, _ := newService(t, 1, libp2p.Options{})

	if err := s1.AddProtocol(newTestProtocol(func(_ context.Context, _ p2p.Peer, _ p2p.Stream) error {
		return nil
	})); err != nil {
		t.Fatal(err)
	}

	addr := serviceUnderlayAddress(t, s1)

	if _, err := s2.Connect(ctx, addr); err != nil {
		t.Fatal(err)
	}

	stream, err := s2.NewStream(ctx, overlay1, nil, testProtocolName, testProtocolVersion, testStreamName)
	if err != nil {
		t.Fatal(err)
	}
	if err := stream.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestNewStream_errNotSupported(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s1, overlay1 := newService(t, 1, libp2p.Options{})

	s2, _ := newService(t, 1, libp2p.Options{})

	addr := serviceUnderlayAddress(t, s1)

	// connect nodes
	if _, err := s2.Connect(ctx, addr); err != nil {
		t.Fatal(err)
	}

	// test for missing protocol
	_, err := s2.NewStream(ctx, overlay1, nil, testProtocolName, testProtocolVersion, testStreamName)
	expectErrNotSupported(t, err)

	// add protocol
	if err := s1.AddProtocol(newTestProtocol(func(_ context.Context, _ p2p.Peer, _ p2p.Stream) error {
		return nil
	})); err != nil {
		t.Fatal(err)
	}

	// test for incorrect protocol name
	_, err = s2.NewStream(ctx, overlay1, nil, testProtocolName+"invalid", testProtocolVersion, testStreamName)
	expectErrNotSupported(t, err)

	// test for incorrect stream name
	_, err = s2.NewStream(ctx, overlay1, nil, testProtocolName, testProtocolVersion, testStreamName+"invalid")
	expectErrNotSupported(t, err)
}

func TestNewStream_semanticVersioning(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s1, overlay1 := newService(t, 1, libp2p.Options{})

	s2, _ := newService(t, 1, libp2p.Options{})

	addr := serviceUnderlayAddress(t, s1)

	if _, err := s2.Connect(ctx, addr); err != nil {
		t.Fatal(err)
	}

	if err := s1.AddProtocol(newTestProtocol(func(_ context.Context, _ p2p.Peer, _ p2p.Stream) error {
		return nil
	})); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		version   string
		supported bool
	}{
		{version: "0", supported: false},
		{version: "1", supported: false},
		{version: "2", supported: false},
		{version: "3", supported: false},
		{version: "4", supported: false},
		{version: "a", supported: false},
		{version: "invalid", supported: false},
		{version: "0.0.0", supported: false},
		{version: "0.1.0", supported: false},
		{version: "1.0.0", supported: false},
		{version: "2.0.0", supported: true},
		{version: "2.2.0", supported: true},
		{version: "2.3.0", supported: true},
		{version: "2.3.1", supported: true},
		{version: "2.3.4", supported: true},
		{version: "2.3.5", supported: true},
		{version: "2.3.5-beta", supported: true},
		{version: "2.3.5+beta", supported: true},
		{version: "2.3.6", supported: true},
		{version: "2.3.6-beta", supported: true},
		{version: "2.3.6+beta", supported: true},
		{version: "2.4.0", supported: false},
		{version: "3.0.0", supported: false},
	} {
		_, err := s2.NewStream(ctx, overlay1, nil, testProtocolName, tc.version, testStreamName)
		if tc.supported {
			if err != nil {
				t.Fatal(err)
			}
		} else {
			expectErrNotSupported(t, err)
		}
	}
}

func TestDisconnectError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s1, overlay1 := newService(t, 1, libp2p.Options{})

	s2, overlay2 := newService(t, 1, libp2p.Options{})

	if err := s1.AddProtocol(newTestProtocol(func(_ context.Context, _ p2p.Peer, _ p2p.Stream) error {
		return p2p.NewDisconnectError(errors.New("test error"))
	})); err != nil {
		t.Fatal(err)
	}

	addr := serviceUnderlayAddress(t, s1)

	if _, err := s2.Connect(ctx, addr); err != nil {
		t.Fatal(err)
	}

	expectPeers(t, s1, overlay2)

	// error is not checked as opening a new stream should cause disconnect from s1 which is async and can make errors in newStream function
	// it is important to validate that disconnect will happen after NewStream()
	_, _ = s2.NewStream(ctx, overlay1, nil, testProtocolName, testProtocolVersion, testStreamName)
	expectPeersEventually(t, s1)
}

const (
	testProtocolName    = "testing"
	testProtocolVersion = "2.3.4"
	testStreamName      = "messages"
)

func newTestProtocol(h p2p.HandlerFunc) p2p.ProtocolSpec {
	return p2p.ProtocolSpec{
		Name:    testProtocolName,
		Version: testProtocolVersion,
		StreamSpecs: []p2p.StreamSpec{
			{
				Name:    testStreamName,
				Handler: h,
			},
		},
	}
}

func expectErrNotSupported(t *testing.T, err error) {
	t.Helper()
	if e := (*p2p.IncompatibleStreamError)(nil); !errors.As(err, &e) {
		t.Fatalf("got error %v, want %T", err, e)
	}
	if !errors.Is(err, multistream.ErrNotSupported) {
		t.Fatalf("got error %v, want %v", err, multistream.ErrNotSupported)
	}
}
