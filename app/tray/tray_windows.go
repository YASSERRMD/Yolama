package tray

import (
	"github.com/YASSERRMD/Yolama/app/tray/commontray"
	"github.com/YASSERRMD/Yolama/app/tray/wintray"
)

func InitPlatformTray(icon, updateIcon []byte) (commontray.OllamaTray, error) {
	return wintray.InitTray(icon, updateIcon)
}
