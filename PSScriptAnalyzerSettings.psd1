# Settings for PSScriptAnalyzer, which the lint job in .github/workflows/ci.yml
# runs on every .ps1 file. Information-level findings (positional parameters)
# are not reported.
@{
    Severity     = @('Error', 'Warning')
    ExcludeRules = @(
        # These are CI, test and packaging scripts that print coloured progress
        # to the console, which is what Write-Host is for.
        'PSAvoidUsingWriteHost',
        # The scripts run on PowerShell 7, which reads UTF-8 without a BOM, and
        # most start with `#!/usr/bin/env pwsh`, which a BOM would break.
        'PSUseBOMForUnicodeEncodedFile'
    )
}
