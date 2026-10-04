package internet

import "syscall"

// controlFunc повторяет сигнатуру бывшего github.com/sagernet/sing
// common/control.Func. Определён в дереве форка, чтобы BeamCore не тянул
// GPL-зависимость sagernet. Используется системным диалером и листенером для
// применения raw-socket контроллеров.
type controlFunc = func(network, address string, c syscall.RawConn) error
