#!/bin/bash
success() {
    COLOR='\033[0;32m'
    RESET='\033[0m'
    printf "${COLOR}%s${RESET}\n" "$1"
}

warn() {
    COLOR='\033[0;33m'
    RESET='\033[0m'
    printf "${COLOR}%s${RESET}\n" "$1"
}

error() {
    COLOR='\033[0;31m'
    RESET='\033[0m'
    printf "${COLOR}%s${RESET}\n" "$1"
}

check() {
    command -v "$1" >/dev/null 2>&1
}

for cmd in wget tar unzip; do
    if ! check "$cmd"; then
        error "Error: $cmd is not installed. Please install it and try again."
        exit 1
    fi
done

OS=$(uname)
case $OS in
'Linux')
    OS='linux'
    ;;
'WindowsNT')
    OS='windows'
    ;;
'Darwin')
    OS='darwin'
    ;;
*)
    OS='unknown'
    ;;
esac

ARCH=$(uname -m)
case $ARCH in
'x86_64')
    ARCH='x86_x64'
    ;;
'i386')
    ARCH='i686'
    ;;
'i686')
    ARCH='i686'
    ;;
'arm64')
    ARCH='aarch64'
    ;;
'aarch64')
    ARCH='aarch64'
    ;;
*)
    ARCH='unknown'
    ;;
esac

echo OS: $OS
if [ $OS == 'unknown' ]; then
    warn "Your OS may not be supported"
fi
echo Arch: $ARCH
if [ $ARCH == 'unknown' ]; then
    warn "Your architecture may not be supported"
fi

default=1

if [ $OS == 'linux' ]; then
    if [ $ARCH == 'x86_x64' ]; then
        default=1
    elif [ $ARCH == 'i686' ]; then
        default=2
    elif [ $ARCH == 'aarch64' ]; then
        default=3
    fi
elif [ $OS == 'windows' ]; then
    if [ $ARCH == 'x86_x64' ]; then
        default=4
    fi
elif [ $OS == 'darwin' ]; then
    if [ $ARCH == 'x86_x64' ]; then
        default=5
    elif [ $ARCH == 'aarch64' ]; then
        default=6
    fi
fi

extension=.tar.gz
echo
echo "Choose version:"
echo "1) linux-x86_x64"
echo "2) linux-i686"
echo "3) linux-aarch64"
echo "4) windows-x86_x64"
echo "5) darwin-x86_x64"
echo "6) darwin-aarch64"
read -p "Your choice (default: $default): " choice
if [ -z "$choice" ]; then
    choice=$default
fi
if [ $choice == '4' ]; then
    extension=.zip
fi

read -p "Download path (default: ./plugin): " path
if [ -z "$path" ]; then
    path=./plugin
fi

mkdir -p $path && cd $path
plugins=("monitor" "agent" "task")
for plugin in "${plugins[@]}"; do
    echo "Downloading $plugin..."
    wget https://github.com/ministruth/$plugin/releases/latest/download/plugin-$OS-$ARCH$extension -O plugin-$OS-$ARCH$extension
    if [ $extension == '.tar.gz' ]; then
        tar -xzf plugin-$OS-$ARCH$extension
    else
        unzip plugin-$OS-$ARCH$extension
    fi
done
mv -f plugin-$OS-$ARCH/* .
rm -rf plugin-$OS-$ARCH$extension plugin-$OS-$ARCH
cd -

echo
success "Download success"
echo Note that you need to put plugin files in your plugin and assets folder:
echo "mv $path/assets/* /path/to/skynet/assets/_plugin && rm -rf $path/assets/"
echo "mv $path/* /path/to/skynet/plugin"
