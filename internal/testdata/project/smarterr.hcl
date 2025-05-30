token "example" {
  position = 2
  source = "parent"
}

hint "hint2" {
  match = {
    error_detail = "parent match"
  }
  suggestion = "parent hint"
}

parameter "param1" {
  value = "parent_value"
}
