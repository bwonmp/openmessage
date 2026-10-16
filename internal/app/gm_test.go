package app

import (
	"strings"
	"testing"

	"go.mau.fi/mautrix-gmessages/pkg/libgm/gmproto"
)

func TestNewContactNumbers(t *testing.T) {
	tests := []struct {
		name   string
		phones []string
		want   []*gmproto.ContactNumber
	}{
		{
			name:   "empty",
			phones: []string{},
			want:   []*gmproto.ContactNumber{},
		},
		{
			name:   "single phone",
			phones: []string{"+1234567890"},
			want: []*gmproto.ContactNumber{
				{MysteriousInt: ContactNumberMysteriousInt, Number: "+1234567890", Number2: "+1234567890"},
			},
		},
		{
			name:   "multiple phones",
			phones: []string{"+1234567890", "9876543210"},
			want: []*gmproto.ContactNumber{
				{MysteriousInt: ContactNumberMysteriousInt, Number: "+1234567890", Number2: "+1234567890"},
				{MysteriousInt: ContactNumberMysteriousInt, Number: "9876543210", Number2: "9876543210"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewContactNumbers(tt.phones)
			if len(got) != len(tt.want) {
				t.Fatalf("NewContactNumbers() len = %v, want %v", len(got), len(tt.want))
			}
			for i := range got {
				if got[i].MysteriousInt != tt.want[i].MysteriousInt ||
					got[i].Number != tt.want[i].Number ||
					got[i].Number2 != tt.want[i].Number2 {
					t.Errorf("NewContactNumbers()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestExtractSIMAndParticipant(t *testing.T) {
	mePayload := &gmproto.SIMPayload{Two: 1}
	otherPayload := &gmproto.SIMPayload{Two: 2}
	fallbackPayload := &gmproto.SIMPayload{Two: 3}

	tests := []struct {
		name              string
		conv              *gmproto.Conversation
		wantParticipantID string
		wantSIM           *gmproto.SIMPayload
	}{
		{
			name: "me participant with sim",
			conv: &gmproto.Conversation{
				Participants: []*gmproto.Participant{
					{IsMe: false, SimPayload: otherPayload},
					{IsMe: true, ID: &gmproto.SmallInfo{Number: "me-id"}, SimPayload: mePayload},
				},
			},
			wantParticipantID: "me-id",
			wantSIM:           mePayload,
		},
		{
			name: "me participant without sim, fallback to simcard",
			conv: &gmproto.Conversation{
				Participants: []*gmproto.Participant{
					{IsMe: true, ID: &gmproto.SmallInfo{Number: "me-id"}},
				},
				SimCard: &gmproto.SIMCard{
					SIMData: &gmproto.SIMData{
						SIMPayload: fallbackPayload,
					},
				},
			},
			wantParticipantID: "me-id",
			wantSIM:           fallbackPayload,
		},
		{
			name: "no me participant, fallback to simcard",
			conv: &gmproto.Conversation{
				Participants: []*gmproto.Participant{
					{IsMe: false, ID: &gmproto.SmallInfo{Number: "other-id"}},
				},
				SimCard: &gmproto.SIMCard{
					SIMData: &gmproto.SIMData{
						SIMPayload: fallbackPayload,
					},
				},
			},
			wantParticipantID: "",
			wantSIM:           fallbackPayload,
		},
		{
			name:              "empty conversation",
			conv:              &gmproto.Conversation{},
			wantParticipantID: "",
			wantSIM:           nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotSIM := ExtractSIMAndParticipant(tt.conv)
			if gotID != tt.wantParticipantID {
				t.Errorf("ExtractSIMAndParticipant() gotID = %v, want %v", gotID, tt.wantParticipantID)
			}
			if gotSIM != tt.wantSIM {
				t.Errorf("ExtractSIMAndParticipant() gotSIM = %v, want %v", gotSIM, tt.wantSIM)
			}
		})
	}
}

func TestBuildSendPayload(t *testing.T) {
	sim := &gmproto.SIMPayload{Two: 1}
	tests := []struct {
		name           string
		conversationID string
		message        string
		replyToID      string
		participantID  string
		sim            *gmproto.SIMPayload
	}{
		{
			name:           "basic text",
			conversationID: "conv1",
			message:        "hello",
			replyToID:      "",
			participantID:  "part1",
			sim:            sim,
		},
		{
			name:           "text with reply",
			conversationID: "conv2",
			message:        "world",
			replyToID:      "msg1",
			participantID:  "part2",
			sim:            sim,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildSendPayload(tt.conversationID, tt.message, tt.replyToID, tt.participantID, tt.sim)
			if got.ConversationID != tt.conversationID {
				t.Errorf("got.ConversationID = %v, want %v", got.ConversationID, tt.conversationID)
			}
			if got.MessagePayload.ConversationID != tt.conversationID {
				t.Errorf("got.MessagePayload.ConversationID = %v, want %v", got.MessagePayload.ConversationID, tt.conversationID)
			}
			if got.MessagePayload.ParticipantID != tt.participantID {
				t.Errorf("got.MessagePayload.ParticipantID = %v, want %v", got.MessagePayload.ParticipantID, tt.participantID)
			}
			if got.SIMPayload != tt.sim {
				t.Errorf("got.SIMPayload = %v, want %v", got.SIMPayload, tt.sim)
			}
			if tt.replyToID != "" {
				if got.Reply == nil || got.Reply.MessageID != tt.replyToID {
					t.Errorf("got.Reply.MessageID = %v, want %v", got.Reply, tt.replyToID)
				}
			} else {
				if got.Reply != nil {
					t.Errorf("got.Reply = %v, want nil", got.Reply)
				}
			}

			msgInfo := got.MessagePayload.MessageInfo[0]
			content := msgInfo.GetMessageContent().GetContent()
			if content != tt.message {
				t.Errorf("got.MessagePayload.MessageInfo[0].MessageContent.Content = %v, want %v", content, tt.message)
			}
		})
	}
}

func TestBuildSendPayloadWithTmpID(t *testing.T) {
	sim := &gmproto.SIMPayload{Two: 1}
	tests := []struct {
		name           string
		conversationID string
		message        string
		replyToID      string
		participantID  string
		sim            *gmproto.SIMPayload
		tmpID          string
	}{
		{
			name:           "with tmpID",
			conversationID: "conv1",
			message:        "hello",
			replyToID:      "",
			participantID:  "part1",
			sim:            sim,
			tmpID:          "custom-tmp-id",
		},
		{
			name:           "empty tmpID generates one",
			conversationID: "conv2",
			message:        "world",
			replyToID:      "msg1",
			participantID:  "part2",
			sim:            sim,
			tmpID:          "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildSendPayloadWithTmpID(tt.conversationID, tt.message, tt.replyToID, tt.participantID, tt.sim, tt.tmpID)

			expectedTmpID := tt.tmpID
			if expectedTmpID == "" {
				if !strings.HasPrefix(got.TmpID, "tmp_") {
					t.Errorf("got.TmpID = %v, want prefix tmp_", got.TmpID)
				}
				expectedTmpID = got.TmpID
			}

			if got.TmpID != expectedTmpID {
				t.Errorf("got.TmpID = %v, want %v", got.TmpID, expectedTmpID)
			}
			if got.MessagePayload.TmpID != expectedTmpID {
				t.Errorf("got.MessagePayload.TmpID = %v, want %v", got.MessagePayload.TmpID, expectedTmpID)
			}
			if got.MessagePayload.TmpID2 != expectedTmpID {
				t.Errorf("got.MessagePayload.TmpID2 = %v, want %v", got.MessagePayload.TmpID2, expectedTmpID)
			}
		})
	}
}

func TestBuildSendMediaPayload(t *testing.T) {
	sim := &gmproto.SIMPayload{Two: 1}
	media := &gmproto.MediaContent{MediaID: "media1"}

	tests := []struct {
		name           string
		conversationID string
		media          *gmproto.MediaContent
		participantID  string
		sim            *gmproto.SIMPayload
	}{
		{
			name:           "basic media",
			conversationID: "conv1",
			media:          media,
			participantID:  "part1",
			sim:            sim,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildSendMediaPayload(tt.conversationID, tt.media, tt.participantID, tt.sim)
			if got.ConversationID != tt.conversationID {
				t.Errorf("got.ConversationID = %v, want %v", got.ConversationID, tt.conversationID)
			}
			if got.MessagePayload.ParticipantID != tt.participantID {
				t.Errorf("got.MessagePayload.ParticipantID = %v, want %v", got.MessagePayload.ParticipantID, tt.participantID)
			}
			if got.SIMPayload != tt.sim {
				t.Errorf("got.SIMPayload = %v, want %v", got.SIMPayload, tt.sim)
			}

			msgInfo := got.MessagePayload.MessageInfo[0]
			gotMedia := msgInfo.GetMediaContent()
			if gotMedia != tt.media {
				t.Errorf("gotMedia = %v, want %v", gotMedia, tt.media)
			}
		})
	}
}

func TestBuildSendMediaPayloadWithTmpID(t *testing.T) {
	sim := &gmproto.SIMPayload{Two: 1}
	media := &gmproto.MediaContent{MediaID: "media1"}

	tests := []struct {
		name           string
		conversationID string
		media          *gmproto.MediaContent
		participantID  string
		sim            *gmproto.SIMPayload
		tmpID          string
	}{
		{
			name:           "with tmpID",
			conversationID: "conv1",
			media:          media,
			participantID:  "part1",
			sim:            sim,
			tmpID:          "custom-media-tmp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildSendMediaPayloadWithTmpID(tt.conversationID, tt.media, tt.participantID, tt.sim, tt.tmpID)

			if got.TmpID != tt.tmpID {
				t.Errorf("got.TmpID = %v, want %v", got.TmpID, tt.tmpID)
			}
		})
	}
}

func TestBuildReactionPayload(t *testing.T) {
	sim := &gmproto.SIMPayload{Two: 1}
	tests := []struct {
		name       string
		messageID  string
		emoji      string
		action     string
		sim        *gmproto.SIMPayload
		wantAction gmproto.SendReactionRequest_Action
	}{
		{
			name:       "add action",
			messageID:  "msg1",
			emoji:      "👍",
			action:     "add",
			sim:        sim,
			wantAction: gmproto.SendReactionRequest_ADD,
		},
		{
			name:       "remove action",
			messageID:  "msg2",
			emoji:      "👎",
			action:     "remove",
			sim:        sim,
			wantAction: gmproto.SendReactionRequest_REMOVE,
		},
		{
			name:       "switch action",
			messageID:  "msg3",
			emoji:      "❤️",
			action:     "switch",
			sim:        sim,
			wantAction: gmproto.SendReactionRequest_SWITCH,
		},
		{
			name:       "default action (unknown maps to add)",
			messageID:  "msg4",
			emoji:      "😂",
			action:     "unknown",
			sim:        sim,
			wantAction: gmproto.SendReactionRequest_ADD,
		},
		{
			name:       "case insensitive action",
			messageID:  "msg5",
			emoji:      "😮",
			action:     "ReMoVe",
			sim:        sim,
			wantAction: gmproto.SendReactionRequest_REMOVE,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildReactionPayload(tt.messageID, tt.emoji, tt.action, tt.sim)
			if got.MessageID != tt.messageID {
				t.Errorf("got.MessageID = %v, want %v", got.MessageID, tt.messageID)
			}
			if got.Action != tt.wantAction {
				t.Errorf("got.Action = %v, want %v", got.Action, tt.wantAction)
			}
			if got.SIMPayload != tt.sim {
				t.Errorf("got.SIMPayload = %v, want %v", got.SIMPayload, tt.sim)
			}
			if got.ReactionData.GetUnicode() != tt.emoji {
				t.Errorf("got.ReactionData.Unicode = %v, want %v", got.ReactionData.GetUnicode(), tt.emoji)
			}
		})
	}
}
