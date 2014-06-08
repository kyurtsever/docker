package server

import (
	"fmt"
	"os"
	"os/exec"
	"log"
	"path"
)

const gcutil = "gcutil"
const pd_mount_base = "/docker-pds"

func idToVolName(imgID string) string {
	return "d-" + string(imgID)[0:60]
}

func idToImgDir(imgID string) string {
	return path.Join(pd_mount_base, idToVolName(imgID))
}

func callCommand(command string, args ...string) error {
	if out, err := exec.Command(command, args...).CombinedOutput(); err != nil {
		return fmt.Errorf("Command \"%v %v\" failed with error %v and output %s", command, args, err, out)
	}
	return nil
}

func detachPd(imgID string) error {
	volName := idToVolName(imgID)
	instanceName, err := os.Hostname()
	if err != nil {
		return err
	}
	diskName := fmt.Sprintf("--device=%s", volName)
        if err := callCommand(gcutil, "detachdisk", "--zone=us-central1-a", diskName, instanceName); err != nil {
		fmt.Printf("Failed to create pd volume %s, err: %s\n", imgID, err)
                return err
        }
        return nil
}

func SaveToPdAndUnattach(imgID string) error {
	pd_path := idToImgDir(imgID)
	log.Printf("Saving imgID:%v to pd_path:%v", imgID, pd_path)
	if err := callCommand("sh", "-c", fmt.Sprintf("docker save %v > %v/%v", imgID, pd_path, imgID)); err != nil {
		return err
	}
	if err := unmontPdImage(imgID); err != nil {
		return err
	}
	return detachPd(imgID);
}

func LoadFromPd(imgID string) error {
	pd_path := idToImgDir(imgID)
	log.Printf("Loading imgID:%v from pd_path:%v", imgID, pd_path)
	return callCommand("sh", "-c", fmt.Sprintf("cat %v/%v | docker load", pd_path, imgID))
}

func pdImageExists(imgID string) (bool, error) {
	volName := idToVolName(imgID)
	fmt.Printf("checking vol %s\n", volName)
        out, err := exec.Command(gcutil, "getdisk", "--zone=us-central1-a", volName).CombinedOutput()
        if err != nil {
		fmt.Printf("pd image does not exist %s", string(out))
                if _, ok := err.(*exec.ExitError); ok {
                        return false, nil
                }
                return false, err
        }
	fmt.Println("pd image exists")
        return true, nil
}

func createPdImage(imgID string) error {
	volName := idToVolName(imgID)
	fmt.Printf("creating vol %s\n", volName)
        if err := callCommand(gcutil, "adddisk", "--zone=us-central1-a", "--size=5", volName); err != nil {
		fmt.Printf("Failed to create pd volume %s, err: %s\n", imgID, err)
                return err
        }
        return nil
}

func attachPdImage(imgID string, mode string) error {
	volName := idToVolName(imgID)
	instanceName, err := os.Hostname()
	if err != nil {
		return err
	}
	diskName := fmt.Sprintf("--disk=%s,mode=%s", volName, mode) 
        if err := callCommand(gcutil, "attachdisk", "--zone=us-central1-a", diskName, instanceName); err != nil {
		fmt.Printf("Failed to create pd volume %s, err: %s\n", imgID, err)
                return err
        }
        return nil
}

func formatAndMount(imgID string) (string, error) {
	volName := idToVolName(imgID)
	devPath := fmt.Sprintf("/dev/disk/by-id/google-%v", volName)
	mountpoint := idToImgDir(imgID)
	if err := os.MkdirAll(mountpoint, 777); err != nil {
		return "", err
	}
	if err := callCommand("/usr/share/google/safe_format_and_mount", "-m", "mkfs.ext4 -F", devPath, mountpoint); err != nil {
		fmt.Printf("Failed to format and mount pd vol %s, err: %s\n", volName, err)
		return "", err
	}
	return mountpoint, nil
}

func unmontPdImage(imgID string) error {
	mountpoint := idToImgDir(imgID)
	if err := callCommand("/bin/umount", mountpoint); err != nil {
		fmt.Printf("Failed to mount pd vol %s, err: %s\n", idToVolName(imgID), err)
		return err
	}
	if err := os.Remove(mountpoint); err != nil {
		return err
	}
	return nil
}

func justMountPdImage(imgID string) (string, error) {
	volName := idToVolName(imgID)
	devPath := fmt.Sprintf("/dev/disk/by-id/google-%v", volName)
	mountpoint := idToImgDir(imgID)
	if err := os.MkdirAll(mountpoint, 777); err != nil {
		return "", err
	}
	if err := callCommand("/bin/mount", "-t", "ext4", devPath, mountpoint); err != nil {
		fmt.Printf("Failed to mount pd vol %s, err: %s\n", volName, err)
		return "", err
	}
	return mountpoint, nil
}

func fastGet(imgID string) error {
	exists, err := pdImageExists(imgID)
	if err != nil {
		return err
	}
	if !exists {
		if err = createPdImage(imgID); err != nil {
			return fmt.Errorf("createPdImage failed: %v", err)
		}
		if err = attachPdImage(imgID, "rw"); err != nil {
			return fmt.Errorf("attachPdImage failed: %v", err)
		}
		if _, err := formatAndMount(imgID); err != nil {
			return fmt.Errorf("formatAndMount failed: %v", err)
		}
		return fmt.Errorf("PD contents weren't prepared yet.")
	} else {
		if err = attachPdImage(imgID, "ro"); err != nil {
			log.Printf("Couldn't attach, perhaps already attached: %v", err)
		}
		if _, err := justMountPdImage(imgID); err != nil {
			log.Printf("Couldn't mount, perhaps already mounted: %v", err)
		}
		if err := LoadFromPd(imgID); err != nil {
			return fmt.Errorf("LoadFromPd failed: %v", err)
		}
		return nil
	}
}
