ethereum {
  key "default" {
    address = "0xafddab345f13d74a35d1e97253e042742faf306d"
	  keystore_path = "./testdata/keystore"
	  passphrase_file = "./testdata/password.txt"
  }
}

secrets {
  foo = {
    "0xafddab345f13d74a35d1e97253e042742faf306d" = "0x04d162904120270fc03e9ad7a0de14f3cf60b7f312087b633fb4fe1a1992135da96692c9fee83e065e66e50a96c550d336065d9cb21203cb4482b61f4211feeae574599cae45a7daab41a33a3f47e929ab790cd7af5fe58df537546e79df3914aa66478e2946243771"
  }
}
