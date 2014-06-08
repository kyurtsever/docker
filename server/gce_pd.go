package server

import (
	"fmt"
	"os/exec"
	"log"
)

func CallCommand(command string, args ...string) error {
	if out, err := exec.Command(command, args...).CombinedOutput(); err != nil {
		return fmt.Errorf("Command \"%v %v\" failed with error %v and output %v", command, args, err, out)
	}
	return nil
}

func SaveToPd(imgID string) error {
	pd_path := fmt.Sprintf("/docker-pds/d-%s", string(imgID)[0:60])
	log.Printf("Saving imgID:%v to pd_path:%v", imgID, pd_path)
	return CallCommand("sh", "-c", fmt.Sprintf("docker save %v > %v/%v", imgID, pd_path, imgID));
}

func LoadFromPd(imgID string) error {
	pd_path := fmt.Sprintf("/docker-pds/d-%s", string(imgID)[0:60])
	log.Printf("Loading imgID:%v from pd_path:%v", imgID, pd_path)
	return CallCommand("sh", "-c", fmt.Sprintf("cat %v/%v | docker load", pd_path, imgID))
}
