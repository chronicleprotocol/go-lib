ethereum {
  key "default" {
    address = "0xafddab345f13d74a35d1e97253e042742faf306d"
	  keystore_path = "./testdata/keystore"
	  passphrase_file = "./testdata/password.txt"
  }
}

secrets {
  foo = {
	  "0xafddab345f13d74a35d1e97253e042742faf306d" = "not a hex value"
  }
}
