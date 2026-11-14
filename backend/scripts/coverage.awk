# Split the union profile without mixing services or test helpers. Duplicate
# blocks emitted by different test binaries contribute their statements once.
NR == 1 { print; next }
{
  file=$1; sub(/:[0-9].*/, "", file)
  belongs=(file ~ ("^ai-buddy/internal/" group "/"))
  if (group == "platform") belongs=belongs || file ~ /^ai-buddy\/cmd\/(migrate|seed)\//
  else if (group == "wiki") belongs=belongs || file ~ /^ai-buddy\/cmd\/wiki(-index)?\//
  else belongs=belongs || file ~ ("^ai-buddy/cmd/" group "/")
  if (belongs) { statements[$1]=$2; if ($3>0) hit[$1]=1 }
}
END {
  for (block in statements) {
    print block, statements[block], (hit[block] ? 1 : 0)
    total+=statements[block]; if (hit[block]) covered+=statements[block]
  }
  if (total==0) { print group ": no coverage blocks" > "/dev/stderr"; exit 1 }
  printf "%s: %.2f%% (%d/%d statements)\n",group,100*covered/total,covered,total > "/dev/stderr"
  if (100*covered<=80*total) exit 1
}
