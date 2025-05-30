token "example" {
  position = 3
  source = "local"
}

token "param_token" {
  position = 4
  source = "parameter"
  parameter = "param1"
}

hint "hint3" {
  match = {
    error_detail = "local match"
  }
  suggestion = "local hint"
}

parameter "param2" {
  value = "local_value"
}
