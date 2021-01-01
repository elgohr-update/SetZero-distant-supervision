package rtc

import (
	"../messages"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc"
	"io"
	"time"
)

type WebRTCStreamer struct {
	peerConnection *webrtc.PeerConnection
	send           chan OutputMessage
	recv           chan []byte
	WebRtcStream   chan  *webrtc.TrackLocalStaticRTP
}

func NewWebRTCStreamer() *WebRTCStreamer {
	connectionInfo, err := createPeerConnection(true)
	if err == nil && connectionInfo != nil {
		rtc := &WebRTCStreamer{send: make(chan OutputMessage, 16384), recv: make(chan []byte, 16384), peerConnection: connectionInfo.peerConnection, WebRtcStream: make(chan *webrtc.TrackLocalStaticRTP)}
		return rtc
	} else {
		return nil
	}
}

func (r *WebRTCStreamer) Send() chan OutputMessage {
	return r.send
}

func (r *WebRTCStreamer) Recv() chan []byte {
	return r.recv
}

func (r *WebRTCStreamer) Start() {
	if _, err := r.peerConnection.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo); err != nil {
		panic(err)
	}

	r.peerConnection.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		go func() {
			ticker := time.NewTicker(time.Second * 3)
			for range ticker.C {
				errSend := r.peerConnection.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: uint32(track.SSRC())}})
				if errSend != nil {
					fmt.Println(errSend)
				}
			}
		}()

		fmt.Printf("Track has started, of type %d: %s \n", track.PayloadType(), track.Codec().MimeType)
		localTrack, newTrackErr := webrtc.NewTrackLocalStaticRTP(track.Codec().RTPCodecCapability, "video", "pion")
		if newTrackErr != nil {
			panic(newTrackErr)
		}
		r.WebRtcStream <-localTrack

		rtpBuf := make([]byte, 1400)
		for {
			i, _, readErr := track.Read(rtpBuf)
			if readErr != nil {
				panic(readErr)
			}

			// ErrClosedPipe means we don't have any subscribers, this is ok if no peers have connected yet
			if _, err := localTrack.Write(rtpBuf[:i]); err != nil && !errors.Is(err, io.ErrClosedPipe) {
				panic(err)
			}
		}

	})

	r.peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("[Streamer] Connection State has changed %s \n", connectionState.String())
		if connectionState == webrtc.ICEConnectionStateFailed ||
			connectionState == webrtc.ICEConnectionStateDisconnected {
			fmt.Println("Done writing media files")
		}
	})

	r.peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		fmt.Println("Canidate: ", candidate)
		if candidate == nil { return }
		iceCandidate, err := json.Marshal(candidate.ToJSON())
		if err != nil || iceCandidate == nil {
			println("Error with Ice canidate")
			return
		}
		r.send <- OutputMessage{Data: iceCandidate, Type: messages.IceCandidate}
	})

	for {
		webRTCMessage := <-r.recv

		var m messageWrapper
		err := json.Unmarshal(webRTCMessage, &m)
		if err == nil {
			switch m.Type {
			case WebRTCOffer:
				fmt.Println("check")
				var offerMessage webRtcOffer
				json.Unmarshal(m.Message, &offerMessage)
				offer := webrtc.SessionDescription{}
				Decode(offerMessage.Offer, &offer)
				err = r.peerConnection.SetRemoteDescription(offer)
				if err != nil {
					fmt.Println("[Streamer] Error: ", err)
				}
				answer, err := r.peerConnection.CreateAnswer(nil)
				if err != nil {
					panic(err)
				}

				jsonAnswer, err := json.Marshal(answer)
				if err != nil {
					panic(err)
				}
				r.send <- OutputMessage{Data: jsonAnswer, Type: messages.AnswerType}

				if err = r.peerConnection.SetLocalDescription(answer); err != nil {
					panic(err)
				}
				break
			case IceCandidate:
				var iceandidate webrtc.ICECandidateInit
				json.Unmarshal(m.Message, &iceandidate)
				err := r.peerConnection.AddICECandidate(iceandidate)
				if err != nil {
					panic(err)
				}
				break
			}
		} else {
			fmt.Println("Error: ", err)
		}
	}
}