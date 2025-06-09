package appaider

import "testing"

func TestLzcdocker(t *testing.T) {
	holder, err := NewLzcDockerHolder()
	if err != nil {
		t.Fatalf("Failed to create docker holder: %v", err)
	}

	// Test with a sample appid
	appid := "cloud.lazycat.app.fiai"
	containers, err := holder.ListContainers(appid)
	if err != nil {
		t.Fatalf("Failed to list containers: %v", err)
	}

	t.Logf("Found %d containers for appid %s:", len(containers), appid)
	for _, c := range containers {
		t.Logf("Container ID: %s, Name: %s, PID: %d", c.ContainerID, c.Name, c.Pid)
	}
}
