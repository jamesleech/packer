if((Test-Path .\build) -ne $true) {
    mkdir build
}

# clean old build
rm .\build\*

# build

