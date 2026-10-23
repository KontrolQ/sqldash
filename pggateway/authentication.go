package pggateway

import (
	"bytes"
	"encoding/binary"
)

func (self *session) authenticate() error {
	remainingBytes := self.gateway.options.MaximumHandshakeBytes
	authenticationStage := 0
	negotiated := false
	for {
		frame, frameError := readFrame(self.backend, self.gateway.options.MaximumHandshakeFrameBytes, &remainingBytes)
		if frameError != nil {
			return frameError
		}
		body := frame.contents[5:]
		switch frame.kind {
		case 'v':
			if negotiated || authenticationStage != 0 || len(body) != 8 {
				return ErrUnsupportedProtocol
			}
			minor := binary.BigEndian.Uint32(body)
			if (minor != 0 && minor != 2) || minor > self.startup.protocol&0xffff || binary.BigEndian.Uint32(body[4:]) != 0 {
				return ErrUnsupportedProtocol
			}
			self.protocol = protocol30 + minor
			negotiated = true
		case 'R':
			if len(body) < 4 {
				return ErrAuthentication
			}
			authenticationType := binary.BigEndian.Uint32(body)
			switch authenticationType {
			case 10:
				if authenticationStage != 0 || !standardSCRAMMechanisms(body[4:]) {
					return ErrChannelBinding
				}
				if writeError := writeAll(self.frontend, frame.contents); writeError != nil {
					return writeError
				}
				if responseError := self.forwardAuthenticationResponse(&remainingBytes, true); responseError != nil {
					return responseError
				}
				authenticationStage = 1
				continue
			case 11:
				if authenticationStage != 1 && authenticationStage != 2 {
					return ErrAuthentication
				}
				if writeError := writeAll(self.frontend, frame.contents); writeError != nil {
					return writeError
				}
				if responseError := self.forwardAuthenticationResponse(&remainingBytes, false); responseError != nil {
					return responseError
				}
				authenticationStage = 2
				continue
			case 12:
				if authenticationStage != 1 && authenticationStage != 2 {
					return ErrAuthentication
				}
				authenticationStage = 3
			case 0:
				if authenticationStage != 3 || len(body) != 4 {
					return ErrAuthentication
				}
				authenticationStage = 4
			default:
				return ErrAuthentication
			}
		case 'K':
			if authenticationStage != 4 {
				return ErrAuthentication
			}
			if keyError := self.storeBackendKey(body); keyError != nil {
				return keyError
			}
		case 'S':
			if authenticationStage != 4 || !validParameterStatus(body) {
				return ErrAuthentication
			}
		case 'N':
			if len(body) == 0 || body[len(body)-1] != 0 {
				return ErrAuthentication
			}
		case 'Z':
			if authenticationStage != 4 || len(body) != 1 || body[0] != 'I' {
				return ErrAuthentication
			}
			self.mutex.Lock()
			if self.backendPID == 0 || self.stopped {
				self.mutex.Unlock()
				return ErrAuthentication
			}
			self.ready = true
			self.mutex.Unlock()
			return writeAll(self.frontend, frame.contents)
		case 'E':
			return ErrAuthentication
		default:
			return ErrAuthentication
		}
		if writeError := writeAll(self.frontend, frame.contents); writeError != nil {
			return writeError
		}
	}
}

func (self *session) forwardAuthenticationResponse(remainingBytes *int, initial bool) error {
	response, responseError := readFrame(self.frontend, self.gateway.options.MaximumHandshakeFrameBytes, remainingBytes)
	if responseError != nil {
		return responseError
	}
	defer clear(response.contents)
	if response.kind != 'p' {
		return ErrAuthentication
	}
	if initial {
		mechanismEnd := bytes.IndexByte(response.contents[5:], 0)
		if mechanismEnd < 0 {
			return ErrAuthentication
		}
		mechanism := string(response.contents[5 : 5+mechanismEnd])
		if mechanism != "SCRAM-SHA-256" {
			return ErrChannelBinding
		}
		body := response.contents[5+mechanismEnd+1:]
		if len(body) < 4 {
			return ErrAuthentication
		}
		initialLength := binary.BigEndian.Uint32(body)
		if initialLength != uint32(len(body)-4) && !(initialLength == ^uint32(0) && len(body) == 4) {
			return ErrAuthentication
		}
	}
	return writeAll(self.backend, response.contents)
}

func standardSCRAMMechanisms(contents []byte) bool {
	return bytes.Equal(contents, []byte("SCRAM-SHA-256\x00\x00"))
}

func validParameterStatus(contents []byte) bool {
	nameEnd := bytes.IndexByte(contents, 0)
	if nameEnd <= 0 || len(contents) < nameEnd+2 || contents[len(contents)-1] != 0 {
		return false
	}
	return bytes.IndexByte(contents[nameEnd+1:len(contents)-1], 0) < 0
}

func (self *session) storeBackendKey(body []byte) error {
	if len(body) < 4+minimumCancelKeyBytes || len(body) > 4+maximumCancelKeyBytes || (self.protocol == protocol30 && len(body) != 8) {
		return ErrAuthentication
	}
	backendPID := binary.BigEndian.Uint32(body)
	if backendPID == 0 {
		return ErrAuthentication
	}
	self.mutex.Lock()
	defer self.mutex.Unlock()
	if self.stopped || self.backendPID != 0 {
		return ErrAuthentication
	}
	self.gateway.mutex.Lock()
	defer self.gateway.mutex.Unlock()
	if _, duplicate := self.gateway.backends[backendPID]; duplicate {
		return ErrAuthentication
	}
	self.backendPID = backendPID
	self.cancelKey = append([]byte(nil), body[4:]...)
	self.gateway.backends[backendPID] = self
	return nil
}
