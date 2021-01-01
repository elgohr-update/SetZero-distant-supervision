package rtc

import (
	"encoding/base64"
	"encoding/json"
	"github.com/pion/webrtc"
)

type WebRtcClient interface {
	Send() chan OutputMessage
	Recv() chan []byte
}

type WebRtcInfo struct {
	peerConnection *webrtc.PeerConnection
	outputTrack    *webrtc.TrackLocalStaticRTP
}

func createPeerConnection(expectStream bool) (*WebRtcInfo, error) {
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	var peerConnection *webrtc.PeerConnection
	var outputTrack *webrtc.TrackLocalStaticRTP
	var err error
	if expectStream {
		m := webrtc.MediaEngine{}
		if err := m.RegisterCodec(webrtc.RTPCodecParameters{
			RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: "video/VP8", ClockRate: 90000, Channels: 0, SDPFmtpLine: "", RTCPFeedback: nil},
			PayloadType:        96,
		}, webrtc.RTPCodecTypeVideo); err != nil {
			panic(err)
		}
		api := webrtc.NewAPI(webrtc.WithMediaEngine(&m))
		peerConnection, _ = api.NewPeerConnection(config)
	} else {
		peerConnection, _ = webrtc.NewPeerConnection(config)
	}


	/**/
	return &WebRtcInfo{peerConnection, outputTrack}, err
}

func Decode(in string, obj interface{}) {
	b, err := base64.StdEncoding.DecodeString(in)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(b, obj)
	if err != nil {
		panic(err)
	}
}
