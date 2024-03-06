package luai

type LuaCheckSum struct {
	Sha256 string `luai:"sha256"`
	Sha512 string `luai:"sha512"`
	Sha1   string `luai:"sha1"`
	Md5    string `luai:"md5"`
}
