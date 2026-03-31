package mdp_helper

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
)

func MergeMpdPlaylist(q *models.MediaQuality, plFile io.Reader) error {
	var masterFile *os.File
	path := filepath.Join(models.OutputPath, q.MediaId.String(), "master.mpd")
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			masterFile, err = os.Create(path)
			if err != nil {
				return err
			}

			defer masterFile.Close()

			_, err = io.Copy(masterFile, plFile)
			if err != nil {
				return err
			}

			return nil
		} else {
			return err
		}
	}

	masterFile, err = os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	defer masterFile.Close()

	masterData, err := io.ReadAll(masterFile)
	if err != nil {
		return err
	}

	masterMpd, err := Unmarshal(masterData)
	if err != nil {
		return err
	}

	plData, err := io.ReadAll(plFile)
	if err != nil {
		return err
	}

	plMpd, err := Unmarshal(plData)
	if err != nil {
		return err
	}

	var index int
	for _, period := range plMpd.Periods {
		for range period.AdaptationSets {
			index++
		}
	}

	for _, profile := range plMpd.Periods {
		for _, adaptationSet := range profile.AdaptationSets {
			adaptationSet.ID = fmt.Sprint(index)
			masterMpd.Periods[0].AdaptationSets = append(
				masterMpd.Periods[0].AdaptationSets,
				adaptationSet,
			)

			index++
		}
	}

	newMasterData, err := Marshal(masterMpd)
	if err != nil {
		return err
	}

	_, err = masterFile.Write(newMasterData)
	if err != nil {
		return err
	}

	return nil
}
