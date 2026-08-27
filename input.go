package main

import (
	"bufio"
	"context"
	"io"
)

func readSessionRequests(ctx context.Context, input io.Reader) <-chan sessionRequest {
	requests := make(chan sessionRequest)
	go func() {
		defer close(requests)
		reader := bufio.NewReader(input)
		for {
			request, err := readSessionRequest(reader)
			if err != nil {
				request = requestQuit
			}
			select {
			case requests <- request:
			case <-ctx.Done():
				return
			}
			if err != nil || request == requestQuit {
				return
			}
		}
	}()
	return requests
}

func readSessionRequest(reader *bufio.Reader) (sessionRequest, error) {
	for {
		character, err := reader.ReadByte()
		if err != nil {
			return 0, err
		}
		switch character {
		case 'q', 3:
			return requestQuit, nil
		case 'r':
			return requestRescan, nil
		case 'f':
			return requestFetch, nil
		case 0x1b:
			request, ok, err := readEscapeRequest(reader)
			if err != nil {
				return 0, err
			}
			if ok {
				return request, nil
			}
		}
	}
}

func readEscapeRequest(reader *bufio.Reader) (sessionRequest, bool, error) {
	prefix, err := reader.ReadByte()
	if err != nil {
		return 0, false, err
	}
	if prefix != '[' {
		return 0, false, nil
	}
	code, err := reader.ReadByte()
	if err != nil {
		return 0, false, err
	}
	switch code {
	case 'A':
		return requestUp, true, nil
	case 'B':
		return requestDown, true, nil
	case '5', '6':
		terminator, err := reader.ReadByte()
		if err != nil {
			return 0, false, err
		}
		if terminator != '~' {
			return 0, false, nil
		}
		if code == '5' {
			return requestPageUp, true, nil
		}
		return requestPageDown, true, nil
	default:
		return 0, false, nil
	}
}
