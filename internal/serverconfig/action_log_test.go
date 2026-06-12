package serverconfig

import (
	"reflect"
	"testing"

	"github.com/pajbot/pajbot2-discord/internal/values"
)

func TestNormalizeActionLogIgnoredChannels(t *testing.T) {
	channelIDs, err := normalizeActionLogIgnoredChannels([]string{" 420 ", "69", "420420420420"})
	if err != nil {
		t.Fatalf("normalizeActionLogIgnoredChannels() error = %v", err)
	}

	want := []string{"420", "69", "420420420420"}
	if !reflect.DeepEqual(channelIDs, want) {
		t.Fatalf("normalizeActionLogIgnoredChannels() = %#v, want %#v", channelIDs, want)
	}
}

func TestNormalizeActionLogIgnoredChannelsRejectsMentions(t *testing.T) {
	if _, err := normalizeActionLogIgnoredChannels([]string{"<#420>"}); err == nil {
		t.Fatal("channel id must be a number only")
	}
}

func TestNormalizeActionLogIgnoredChannelsRejectsInvalidChannel(t *testing.T) {
	if _, err := normalizeActionLogIgnoredChannels([]string{"abc"}); err == nil {
		t.Fatal("channel id must be a number only")
	}
}

func TestActionLogChannelIgnored(t *testing.T) {
	mutex.Lock()
	data = map[string]string{}
	mutex.Unlock()
	t.Cleanup(func() {
		mutex.Lock()
		data = map[string]string{}
		mutex.Unlock()
	})

	set("forsen", "value:"+values.ActionLogIgnoredChannels, "420,69")

	if ActionLogChannelIgnored("forsen", "420") {
		t.Fatal("420 must be ignored")
	}

	if ActionLogChannelIgnored("forsen", "69") {
		t.Fatal("69 must be ignored")
	}

	if !ActionLogChannelIgnored("forsen", "123") {
		t.Fatal("123 must not be ignored")
	}
}
