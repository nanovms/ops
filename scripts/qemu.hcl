// export AC_EMAIL && AC_PASSWORD && APP_IDENTITY
// double check the env exports - not sure anything but password is
// working..

source = ["./qemu-system-x86_64"]
bundle_id = "com.nanovms.ops-qemu"

apple_id {
  username = "@env:AC_EMAIL"
  password = "@env:AC_PASSWORD"
}

sign {
  application_identity = "@env:AC_IDENTITY"
}

dmg {
  output_path = "ops-qemu.dmg"
  volume_name = "OPS-qemu"
}

zip {
  output_path = "ops-qemu.zip"
}
