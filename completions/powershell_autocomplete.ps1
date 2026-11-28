# A native completer gets the word, the command line and the cursor, in that order,
# so $wordToComplete below holds the whole command line, which runs again with the
# completion flag. $commandName and $cursorPosition only keep the positions.
[Diagnostics.CodeAnalysis.SuppressMessageAttribute('PSReviewUnusedParameter', '')]
[Diagnostics.CodeAnalysis.SuppressMessageAttribute('PSAvoidUsingInvokeExpression', '')]
param()

$fn = $($MyInvocation.MyCommand.Name)
$name = $fn -replace "(.*)\.ps1$", '$1'
Register-ArgumentCompleter -Native -CommandName $name -ScriptBlock {
     param($commandName, $wordToComplete, $cursorPosition)
     $other = "$wordToComplete --generate-shell-completion"
         Invoke-Expression $other | ForEach-Object {
            [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
         }
 }
