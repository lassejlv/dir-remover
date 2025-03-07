const dev_path = Deno.args[0]

if (!dev_path) {
  console.error('Please provide a path')
  Deno.exit(1)
}

const continue_response = confirm(`Are you sure you wanna continue in '${dev_path}'?`)
if (!continue_response) {
  console.log('Aborting')
  Deno.exit(0)
}

function getDirs(path: string) {
  return Deno.readDir(path)
}

async function askToDelete(dirs: AsyncIterable<Deno.DirEntry>): Promise<string[]> {
  const to_delete: string[] = []

  for await (const dir of dirs) {
    const response = confirm(`Do you want to delete ${dir.name}?`)
    if (response) {
      to_delete.push(dir.name)
    }
  }

  return to_delete
}

async function deleteDirs(directories: string[]) {
  if (directories.length === 0) {
    console.log('Nothing to delete')
    return
  }

  console.log(`${directories.join(', ')} will be deleted`)

  const confirmed = confirm('Are you sure you want to delete these directories?')
  if (!confirmed) {
    console.log('Aborting')
    return
  }

  for (const dir of directories) {
    try {
      await Deno.remove(`${dev_path}/${dir}`, { recursive: true })
      console.log(`Deleted ${dir}`)
    } catch (error) {
      console.error(`Failed to delete ${dir}: ${error}`)
    }
  }
}

async function main() {
  const dirs = getDirs(dev_path)
  const to_delete = await askToDelete(dirs)
  await deleteDirs(to_delete)
}

main()
