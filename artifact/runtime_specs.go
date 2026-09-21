package artifact

type runtimeSpec struct {
	url           string
	sha256        string
	bytes         int64
	member        string
	libraryName   string
	librarySHA256 string
	libraryBytes  int64
}

var runtimes = map[string]runtimeSpec{
	"darwin/arm64": newRuntimeSpec(
		"onnxruntime-osx-arm64-1.23.2.tgz",
		"b4d513ab2b26f088c66891dbbc1408166708773d7cc4163de7bdca0e9bbb7856", 9_999_931,
		"onnxruntime-osx-arm64-1.23.2/lib/libonnxruntime.1.23.2.dylib",
		"libonnxruntime.dylib", "d306d2bc768540766c7ed8a1e0ff05d2870c77a934ebeee4a7bafa1b732ef299", 35_138_784,
	),
	"darwin/amd64": newRuntimeSpec(
		"onnxruntime-osx-x86_64-1.23.2.tgz",
		"d10359e16347b57d9959f7e80a225a5b4a66ed7d7e007274a15cae86836485a6", 11_676_322,
		"onnxruntime-osx-x86_64-1.23.2/lib/libonnxruntime.1.23.2.dylib",
		"libonnxruntime.dylib", "8c9c78de65ea3786f987c0d980e9c1b13a3a5fbc6b3e2965ba05b450e6e4c054", 39_742_608,
	),
	"linux/arm64": newRuntimeSpec(
		"onnxruntime-linux-aarch64-1.23.2.tgz",
		"7c63c73560ed76b1fac6cff8204ffe34fe180e70d6582b5332ec094810241e5c", 7_254_068,
		"onnxruntime-linux-aarch64-1.23.2/lib/libonnxruntime.so.1.23.2",
		"libonnxruntime.so", "648ffa64fbe027ae27139109410900cf776a030dec2dbbac51053318cc44c286", 18_693_384,
	),
	"linux/amd64": newRuntimeSpec(
		"onnxruntime-linux-x64-1.23.2.tgz",
		"1fa4dcaef22f6f7d5cd81b28c2800414350c10116f5fdd46a2160082551c5f9b", 8_309_231,
		"onnxruntime-linux-x64-1.23.2/lib/libonnxruntime.so.1.23.2",
		"libonnxruntime.so", "13ab8084954fa4a47c777880180b90810d6020f021441395712b48a75b74c68b", 22_326_072,
	),
}

func newRuntimeSpec(archive, archiveHash string, archiveBytes int64, member, library, libraryHash string, libraryBytes int64) runtimeSpec {
	return runtimeSpec{
		url:           "https://github.com/microsoft/onnxruntime/releases/download/v" + ONNXRuntimeVersion + "/" + archive,
		sha256:        archiveHash,
		bytes:         archiveBytes,
		member:        member,
		libraryName:   library,
		librarySHA256: libraryHash,
		libraryBytes:  libraryBytes,
	}
}
