variables {
  a = var.b
  b = var.e
  c = "hello"
  d = "world"
  e = "${var.c} ${var.d}"

  aa = [
    var.bb1,
    var.bb2,
    var.bb3,
  ]
  bb = var.aa
  bb1 = 1
  bb2 = 2
  bb3 = 3

  aaa = {
    bbb = 4
  }
  bbb = var.aaa.bbb
}
