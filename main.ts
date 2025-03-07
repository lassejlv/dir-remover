import consola from 'consola'

let dev_path = Deno.args[0]

if (dev_path === '.') {
  dev_path = Deno.cwd()
} else if (dev_path === undefined) {
  dev_path = Deno.cwd()
}

if (!dev_path) {
  consola.error('No path provided')
  Deno.exit(1)
}

const continue_response = confirm(`Are you sure you wanna continue in '${dev_path}'?`)
if (!continue_response) {
  consola.info('Aborting')
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
    consola.info('Nothing to delete')
    return
  }

  console.info(`${directories.join(', ')} will be deleted`)

  const confirmed = confirm('Are you sure you want to delete these directories?')
  if (!confirmed) {
    consola.info('Aborting')
    return
  }

  for (const dir of directories) {
    try {
      await Deno.remove(`${dev_path}/${dir}`, { recursive: true })
      consola.success(`Deleted ${dir}`)
    } catch (error) {
      consola.error(`Failed to delete ${dir}: ${error}`)
    }
  }
}

const dirs = getDirs(dev_path)
const to_delete = await askToDelete(dirs)
await deleteDirs(to_delete)
