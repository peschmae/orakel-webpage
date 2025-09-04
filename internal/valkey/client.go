package valkey

import (
	"context"
	"encoding/json"
	"strings"

	valkey "github.com/valkey-io/valkey-go"
)

type ValkeyClient struct {
	valkey.Client
}

func NewValKeyClient(valkeyHost, valkeyPort string) (*ValkeyClient, error) {

	client, err := valkey.NewClient(valkey.ClientOption{InitAddress: []string{valkeyHost + ":" + valkeyPort}})
	if err != nil {
		return nil, err
	}

	// Ensure the client is connected
	if err = client.Do(context.TODO(), client.B().Ping().Build()).Error(); err != nil {
		return nil, err
	}

	return &ValkeyClient{client}, nil
}

func (v *ValkeyClient) getEntry(ctx context.Context, key string) (string, error) {

	res, err := v.Client.Do(ctx, v.Client.B().Get().Key(strings.ToLower(key)).Build()).AsBytes()
	if err != nil {
		return "", err
	}

	if len(res) == 0 {
		return "", nil // No entry found for the given key
	}
	return string(res), nil
}

func (v ValkeyClient) GetRecording(ctx context.Context, keySuffix string) (*WorkloadRecording, error) {

	key := "recording:" + keySuffix

	value, err := v.getEntry(ctx, key)
	if err != nil {
		return nil, err
	}
	if value == "" {
		return nil, nil // No recording found for the given key
	}

	var workloadRecording WorkloadRecording
	err = json.Unmarshal([]byte(value), &workloadRecording)
	if err != nil {
		return nil, err
	}
	return &workloadRecording, nil

}
