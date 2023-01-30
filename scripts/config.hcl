// export AC_EMAIL && AC_PASSWORD && APP_IDENTITY

source = ["./ops"]
bundle_id = "com.nanovms.ops"

apple_id {
  username = "@env:AC_EMAIL"
  password = "@env:AC_PASSWORD"
}

sign {
  application_identity = "@env:APP_IDENTITY"
}

dmg {
  output_path = "ops.dmg"
  volume_name = "OPS"
}

zip {
  output_path = "ops.zip"
}
