New-Module -name circonus-install -ScriptBlock {
  $installpath = "${env:systemdrive}\Program Files\Circonus\Circonus-Unified-Agent"
  $name = "circonus-unified-agent"
  $repo = "circonus-labs/${name}"
  $releases = "https://api.github.com/repos/${repo}/releases"
  $zip = "${name}.zip"

  function Get-Win32OS {
    if(!$global:win32OS)
    {
      $global:win32OS = Get-WMIQuery win32_operatingsystem
    }
    $global:win32OS
  }

  function New-Location {
    if (!(Test-Path $installpath)) {
      New-Item -ItemType Directory -Force -Path $installpath
    } else {
      return
    }
  }

  function Get-Latest-Release {
    Write-Host "Determining latest release..."
    $tag = (Invoke-WebRequest $releases | ConvertFrom-Json)[0].tag_name
    $tagrawv = $tag.substring(1)
    $download = "https://github.com/${repo}/releases/download/${tag}/circonus-unified-agent_${tagrawv}_windows_x86_64.zip"
    return $download
  }

  function Get-Package {
    param ($downloadpath)
    Write-Host "Downloading Package..."
    Invoke-WebRequest $downloadpath -Out "${env:temp}\${zip}"
  }

  function Expand-Package {
    Write-Host "Expanding archive and installing..."
    Expand-Archive -Path "${env:temp}\${zip}" -DestinationPath "${env:systemdrive}\Program Files\Circonus\Circonus-Unified-Agent"
  }

  function Enable-Service {
    Write-Host ".........."
    & "${installpath}\sbin\${name}d.exe" --service install --config "${installpath}\etc\circonus-unified-agent.conf"
    Set-Service -Name circonus-unified-agent -StartupType Automatic
  }

  function Set-Config {
    param ($token)
    Write-Host "Copying config..."
    Move-Item -Path "${installpath}\etc\example-circonus-unified-agent_windows.conf" -Destination "${installpath}\etc\circonus-unified-agent.conf"
    $file = "${installpath}\etc\circonus-unified-agent.conf"
    (Get-Content $file) -replace '  api_token = ""', "  api_token = `"${token}`"" | Set-Content $file
  }

  function Cleanup {
    Write-Host "Cleaning up..."
    Remove-Item -Path "${env:temp}\${zip}" -Force
  }

  function Start-Service {
    Write-Host "Starting service..."
    Set-Service -Name circonus-unified-agent -Staus Running -PassThru
  }

  function Install-Project {
    param (
      [string]$key = ""
    )
    if ($key -eq "" ) {
      Write-Host "Circonus API Key is required."
      exit
    }
    if ((Test-Path $installpath)) {
      Write-Host "Circonus-Unified-Agent is already installed."
      exit
    }
    if ((Get-Win32OS).osarchitecture -notlike '64-bit') {
      Write-Host "Circonus-Unified-Agent is only supported on 64-Bit Windows releases."
    }

    # Create the install directory
    New-Location
    # Determine the latest release
    $release = Get-Latest-Release
    # Fetch the latest CUA version zip file
    Get-Package($release)
    # Unarchive the zip file into their proper location
    Expand-Package
    # Set the service up
    Enable-Service
    # Setup the default configuration file
    Set-Config($key)
    # Cleanup tmp dir
    Cleanup
    # Start the service
    Start-Service

  }
  Set-Alias install -value Install-Project
  Export-ModuleMember -function 'Install-Project' -alias 'install'
}