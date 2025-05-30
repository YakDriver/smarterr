token "example" {
  position = 1
  source = "hint"
}

hint "hint1" {
  match = {
    error_detail = "global match"
  }
  suggestion = "global hint"
}

parameter "param1" {
  value = "global_value"
}
