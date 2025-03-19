import consola from 'consola'
import denoJson from "./deno.json" with { type: "json"}

const cmd = Deno.args[0]

if (cmd === '--help' || cmd === '-h') {
  console.log(`
  Usage: dir-remover [path]
  Options:
    --all    Skip individual confirmations and ask to delete all files at once
  `)
  Deno.exit(0)
}

if (cmd === '--version' || cmd === '-v') {
  console.log(denoJson.version)
  Deno.exit(0)
}

const allFlag = Deno.args.includes('--all')
let dev_path = Deno.args.filter(arg => arg !== '--all')[0]

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

async function getAllDirs(dirs: AsyncIterable<Deno.DirEntry>): Promise<string[]> {
  const allDirs: string[] = []
  
  for await (const dir of dirs) {
    allDirs.push(dir.name)
  }
  
  return allDirs
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

let to_delete: string[]
if (allFlag) {
  to_delete = await getAllDirs(dirs)
} else {
  to_delete = await askToDelete(dirs)
}

await deleteDirs(to_delete)
