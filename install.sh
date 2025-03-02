#!/bin/sh

# This install script is intended to download and install the latest available
# release of the ops.
# Installer script inspired from:
#  1) https://wasmer.io/
#  2) https://sh.rustup.rs
#  3) https://yarnpkg.com/install.sh
#  4) https://raw.githubusercontent.com/brainsik/virtualenv-burrito/master/virtualenv-burrito.sh
#
# It attempts to identify the current platform and an error will be thrown if
# the platform is not supported.
#
# Environment variables:
# - INSTALL_DIRECTORY (optional): defaults to $HOME/.ops

reset="\033[0m"
red="\033[31m"
green="\033[32m"
yellow="\033[33m"
cyan="\033[36m"
white="\033[37m"
bold="\e[1m"
dim="\e[2m"

RELEASES_URL="https://storage.googleapis.com/cli"
REPOSITORY_URL="https://raw.githubusercontent.com/nanovms/ops/master"


initArch() {
    ARCH=$(uname -m)
}

initOS() {
    OS=$(uname | tr '[:upper:]' '[:lower:]')
    if [ -n "$OPS_OS" ]; then
        printf "$cyan> Using OPS_OS ($OPS_OS).$reset\n"
        OS="$OPS_OS"
    fi
    case "$OS" in
        darwin) OS='darwin';;
        linux) OS='linux';;
        *) printf "$red> The OS (${OS}) is not supported ops.$reset\n"; exit 1;;
    esac
}

download_file() {
    url="$1"
    destination="$2"

    echo "Fetching $url.."
    if test -x "$(command -v curl)"; then
        code=$(curl --progress-bar -w '%{http_code}' -L "$url" -o "$destination")
    elif test -x "$(command -v wget)"; then
        code=$(wget --show-progress --progress=bar:force:noscroll -q -O "$destination" --server-response "$url" 2>&1 | awk '/^  HTTP/{print $2}' | tail -1)
    else
        printf "$red> Neither curl nor wget was available to perform http requests.$reset\n"
        exit 1
    fi

    if [ "$code" != 200 ]; then
        printf "$red>File download failed with code $code.$reset\n"
        exit 1
    fi
}

ops_download() {
  # determine install directory if required
  if [ -z "$INSTALL_DIRECTORY" ]; then
      INSTALL_DIRECTORY="$HOME/.ops"
  fi
  OPS=INSTALL_DIRECTORY

  # TODO: Track release TAGS and update.
  # use github release tags

  # assemble expected release URL

  # Download ops
  if [ "$ARCH" = "aarch64" ]; then
    BINARY_URL="$RELEASES_URL/${OS}/aarch64/ops"
  elif [ "$ARCH" = "arm64" ]; then
    BINARY_URL="$RELEASES_URL/${OS}/aarch64/ops"
  else 
    BINARY_URL="$RELEASES_URL/${OS}/ops"
  fi

  DOWNLOAD_FILE=$(mktemp -t ops.XXXXXXXXXX)

  download_file "$BINARY_URL" "$DOWNLOAD_FILE"
  printf "\033[2A$cyan> Downloading latest release... ✓$reset\033[K\n"
  chmod +x "$DOWNLOAD_FILE"

  INSTALL_NAME="ops"
  mkdir -p $INSTALL_DIRECTORY/bin
  mv "$DOWNLOAD_FILE" "$INSTALL_DIRECTORY/bin/$INSTALL_NAME"

  # Download bash completion script
  DOWNLOAD_FILE=$(mktemp -t bash_completion.XXXXXXXXXX)
  download_file "${REPOSITORY_URL}/bash_completion.sh" "$DOWNLOAD_FILE"
  printf "\033[K\n\033[1A"
  chmod +x "$DOWNLOAD_FILE"

  INSTALL_PATH="${INSTALL_DIRECTORY}/scripts/bash_completion.sh"
  mkdir -p "${INSTALL_DIRECTORY}/scripts"
  mv "$DOWNLOAD_FILE" "$INSTALL_PATH"
}


ops_detect_profile() {
  if [ -n "${PROFILE}" ] && [ -f "${PROFILE}" ]; then
    echo "${PROFILE}"
    return
  fi

  local DETECTED_PROFILE
  DETECTED_PROFILE=''
  local SHELLTYPE
  SHELLTYPE="$(basename "/$SHELL")"

  if [ "$SHELLTYPE" = "bash" ]; then
    if [ -f "$HOME/.bashrc" ]; then
      DETECTED_PROFILE="$HOME/.bashrc"
    elif [ -f "$HOME/.bash_profile" ]; then
      DETECTED_PROFILE="$HOME/.bash_profile"
    fi
  elif [ "$SHELLTYPE" = "zsh" ]; then
    DETECTED_PROFILE="$HOME/.zshrc"
  elif [ "$SHELLTYPE" = "fish" ]; then
    DETECTED_PROFILE="$HOME/.config/fish/config.fish"
  fi

  if [ -z "$DETECTED_PROFILE" ]; then
    if [ -f "$HOME/.profile" ]; then
      DETECTED_PROFILE="$HOME/.profile"
    elif [ -f "$HOME/.bashrc" ]; then
      DETECTED_PROFILE="$HOME/.bashrc"
    elif [ -f "$HOME/.bash_profile" ]; then
      DETECTED_PROFILE="$HOME/.bash_profile"
    elif [ -f "$HOME/.zshrc" ]; then
      DETECTED_PROFILE="$HOME/.zshrc"
    elif [ -f "$HOME/.config/fish/config.fish" ]; then
      DETECTED_PROFILE="$HOME/.config/fish/config.fish"
    fi
  fi

  if [ ! -z "$DETECTED_PROFILE" ]; then
    echo "$DETECTED_PROFILE"
  fi
}

ops_detect_supported_linux_distribution() {
  if [ -f /etc/os-release ]; then
    . /etc/os-release
    DETECTED_DISTRIBUTION=$NAME
  elif type lsb_release >/dev/null 2>&1; then
    DETECTED_DISTRIBUTION=$(lsb_release -si)
  elif [ -f /etc/lsb-release ]; then
    . /etc/lsb-release
    DETECTED_DISTRIBUTION=$DISTRIB_ID
  elif [ -f /etc/debian_version ]; then
    DETECTED_DISTRIBUTION=debian
  elif [ -f /etc/fedora-release ]; then
    DETECTED_DISTRIBUTION=fedora
  elif [ -f /etc/centos-release ]; then
    DETECTED_DISTRIBUTION=centos
  fi

  echo "$DETECTED_DISTRIBUTION"
}

ops_brew_install_qemu() {
  if which brew >/dev/null; then
    brew install qemu
  else
    printf "Homebrew not found.Please install from https://brew.sh/"
  fi
}

ops_apt_install_qemu(){
  apt install qemu-system-x86 -y --no-upgrade
}

ops_pacman_install_qemu(){
  pacman -S qemu-system-x86
}

ops_dnf_install_qemu(){
  dnf install qemu-kvm qemu-img -y
}

ops_yum_install_qemu(){
  yum install qemu-kvm qemu-img -y
}

ops_install_qemu() {
  if which qemu-system-x86_64>/dev/null; then
    return
  fi
  # install qemu on mac or supported linux distributions
  if [ "$OS" = "darwin" ]; then
    ops_brew_install_qemu
  else
    LINUX_DISTRIBUTION=`echo $(ops_detect_supported_linux_distribution) | tr '[:upper:]' '[:lower:]'`
    case "$LINUX_DISTRIBUTION" in
      *ubuntu*)
        ops_apt_install_qemu
        ;;
      *fedora*)
        ops_dnf_install_qemu
        ;;
      *centos*)
        ops_yum_install_qemu
        ;;
      *debian*)
        ops_apt_install_qemu
        ;;
      *arch linux*)
        ops_pacman_install_qemu
        ;;
    esac
  fi

  if ! which qemu-system-x86_64>/dev/null; then
    printf "QEMU not found. Please install QEMU using your package manager and re-run this script"
  fi
}

ops_user_permissions() {
    local group="kvm"
    local username=`whoami`
    local prompt="Ops uses kvm acceleration. Would you like to add ${username} to the kvm group [Yes/no]? "
    local notification="Adding ${username} to ${group} using sudo."
    local failure="Unable to add ${username} to ${group}. Please add the $username to ${group} manually."
    local success="${username} has been added to kvm ${group}. The change will take affect on the next login."
    local add_user

    if groups $username | grep -q "\b${group}\b"; then
        return
    fi

    while true
    do
        printf "$prompt"
        read REPLY
        local reply=`echo $REPLY | tr '[:upper:]' '[:lower:]'`

        if [ -z "$reply" ]; then
            reply="yes"
        fi

        case "$reply" in
            no) add_user=false
                break
                ;;
            yes) add_user=true
                 break
                 ;;
            *) echo "Valid responses are either \"yes\" or \"no\"".
        esac
    done


    if ! $add_user; then
        return
    fi

    printf "$notification\n"
    sudo usermod -aG $group $username

    if [ $? -eq 0 ]; then
        printf "$success\n"
    else
        printf "$failure\n"
    fi
}


ops_set_user_permissions(){
    if [ "$OS" = "linux" ]; then
        ops_user_permissions
    fi
}

ops_link() {
  printf "$cyan> Adding to bash profile...$reset\n"
  OPS_PROFILE="$(ops_detect_profile)"

  SHELLTYPE="$(basename "/$SHELL")"

  if [ "$SHELLTYPE" = "zsh" ]; then
    SOURCE_STR="# OPS config\nexport OPS_DIR=\"\$HOME/.ops\"\nexport PATH=\"\$HOME/.ops/bin:\$PATH\"\nsource \"\$HOME/.ops/scripts/bash_completion.sh\"\nautoload bashcompinit\n"
  else
    SOURCE_STR="# OPS config\nexport OPS_DIR=\"\$HOME/.ops\"\nexport PATH=\"\$HOME/.ops/bin:\$PATH\"\nsource \"\$HOME/.ops/scripts/bash_completion.sh\"\n"
  fi

  echo "------------- ${OPS_PROFILE}"

  # Create the ops.sh file
  echo "$SOURCE_STR" > "$HOME/.ops/ops.sh"

  if [ -z "${OPS_PROFILE-}" ] ; then
    printf "${red}Profile not found. Tried:\n* ${OPS_PROFILE} (as defined in \$PROFILE)\n* ~/.bashrc\n* ~/.bash_profile\n* ~/.zshrc\n* ~/.profile.\n"
    echo "\nHow to solve this issue?\n* Create one of them and run this script again"
    echo "* Create it (touch ${OPS_PROFILE}) and run this script again"
    echo "  OR"
    printf "* Append the following lines to the correct file yourself:$reset\n"
    command printf "${SOURCE_STR}"
  else
    if ! grep -q 'ops' "$OPS_PROFILE"; then
      command printf "$SOURCE_STR" >> "$OPS_PROFILE"
    fi
    printf "\033[1A$cyan> Adding to bash profile... ✓$reset\n"
    printf "${dim}Note: We've added the following to your $OPS_PROFILE\n"
    echo "If this isn't the profile of your current shell then please add the following to your correct profile:"
    printf "   $SOURCE_STR$reset\n"

    version=`$HOME/.ops/bin/ops version` || (
      printf "$red> ops was installed, but doesn't seem to be working :($reset\n"
      exit 1;
    )
    printf "$green> Successfully installed ops $version! Please open another terminal where the \`ops\` command will now be available.$reset\n"
  fi
}

checkHWAccelSupport() {
  echo ""

  local hwSupported=false
  local haveRights=false

  if [ "$OS" = "linux" ]; then
    local group="kvm"
    local username=`whoami`
    local hw_support=`grep -woE 'svm|vmx' /proc/cpuinfo | uniq`

    #check acceleration supported
    if [ "$hw_support" = "svm" ] || [ "$hw_support" = "vmx" ]; then
      hwSupported=true
    fi

    #check permissions
    if groups $username | grep -q "\b${group}\b"; then
        haveRights=true
    fi
  fi

  if [ "$OS" = "darwin" ]; then
    haveRights=true
    local hw_support=`sysctl kern.hv_support`
    if [ "$hw_support" = "kern.hv_support: 1" ]; then
      hwSupported=true
    fi
  fi

  if [ "$hwSupported" = "false" ]; then
        echo "$yellow> Hardware acceleration not supported by your system. $reset"
        echo "$yellow> Ops will attempt to enable acceleration by default and will show a warning if the system doesn't support it. $reset"
        echo "$yellow> To avoid such warnings you may disable acceleration in configuration or via command line parameters. $reset"
  else
    if [ "$haveRights" = "false" ]; then
      echo "$yellow> Hardware acceleration is supported, but you don't have rights. $reset"
      echo "$yellow> Try adding yourself to the kvm group: sudo adduser ${username} kvm $reset"
      echo "$yellow> You'll need to re-login for this to take effect. $reset"
    fi
  fi
}

ops_update() {
  if [ "$ARCH" = "aarch64" ]; then
    "$HOME/.ops/bin/ops" update --arm
  elif [ "$ARCH" = "arm64" ]; then
    "$HOME/.ops/bin/ops" update --arm
  else
    "$HOME/.ops/bin/ops" update
  fi
}

ops_install() {
  magenta1="${reset}\033[34;1m"
  magenta2="${reset}\033[34m"
  magenta3="${reset}\033[34;2m"

  if which ops >/dev/null; then
    printf "${reset}Updating ops$reset\n"
  else
    printf "${reset}Installing ops!$reset\n"
  fi

  # identify platform based on uname output
  initOS
  initArch
  ops_install_qemu
  ops_set_user_permissions
  checkHWAccelSupport
  ops_download
  ops_link
  ops_update
}

ops_install
