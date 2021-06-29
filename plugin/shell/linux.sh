#!/bin/bash

case $1 in
install)
    set -e
    echo "Installing gotty..."
    tar -xzf ./plugin/24a3568a-1147-4f0b-8810-0eac68a7600b/*.tar.gz -C ./plugin/24a3568a-1147-4f0b-8810-0eac68a7600b/
    rm ./plugin/24a3568a-1147-4f0b-8810-0eac68a7600b/*.tar.gz
    chmod +x ./plugin/24a3568a-1147-4f0b-8810-0eac68a7600b/gotty
    echo "Install success"
    ;;
uninstall)
    echo "Uninstalling gotty..."
    rm -rf ./plugin/24a3568a-1147-4f0b-8810-0eac68a7600b/
    echo "Uninstall success"
    ;;
*)
    echo "Operation invalid"
    exit 1
    ;;
esac
