package generator

type Triple struct {
	arch _arch
	os   _os
	env  _env
}

func NewTriple(a _arch, o _os, e _env) *Triple {
	return &Triple{arch: a, os: o, env: e}
}

func (t *Triple) String() string {
	return t.arch.String() + "-" + t.os.String() + "-" + t.env.String()
}

type _arch uint8

const (
	AARCH64 _arch = iota
	X86_64
)

func (a _arch) String() string {
	switch a {
	case AARCH64:
		return "aarch64"
	case X86_64:
		return "x86_64"
	}
	return ""
}

type _os uint8

const (
	LINUX _os = iota
	DARWIN
	WINDOWS
)

func (o _os) String() string {
	switch o {
	case LINUX:
		return "linux"
	case DARWIN:
		return "darwin"
	case WINDOWS:
		return "windows"
	}
	return ""
}

type _env uint8

const (
	UNKNOWN _env = iota
)

func (e _env) String() string {
	switch e {
	case UNKNOWN:
		return "unknown"
	}
	return ""
}
