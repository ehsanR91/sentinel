import fs from 'fs'
import path from 'path'
import sharp from 'sharp'

const root = path.resolve()
const publicDir = path.join(root, 'public')
const iconsDir = path.join(publicDir, 'icons')
const sourceSvgPath = path.join(iconsDir, 'source.svg')
const sourceMonoSvgPath = path.join(iconsDir, 'source-monochrome.svg')

const ensureDir = async (dir) => {
  await fs.promises.mkdir(dir, { recursive: true })
}

const writeFile = async (filePath, content) => {
  await fs.promises.writeFile(filePath, content, 'utf8')
}

const svgSource = await fs.promises.readFile(sourceSvgPath)
const svgMonoSource = await fs.promises.readFile(sourceMonoSvgPath)

await ensureDir(iconsDir)

const iconSizes = [16, 32, 48, 36, 72, 96, 144, 192, 256, 384, 512, 1024]
const appleSizes = [57, 60, 72, 76, 114, 120, 144, 152, 167, 180]
const msSpecs = [
  { name: 'mstile-70x70.png', width: 70, height: 70 },
  { name: 'mstile-150x150.png', width: 150, height: 150 },
  { name: 'mstile-310x150.png', width: 310, height: 150 },
  { name: 'mstile-310x310.png', width: 310, height: 310 }
]
const maskableSizes = [36, 48, 72, 96, 144, 192, 512, 1024]
const monoSizes = [192, 512]
const splashSpecs = [
  { name: 'apple-splash-640x1136.png', width: 640, height: 1136, media: '(device-width: 320px) and (device-height: 568px) and (-webkit-device-pixel-ratio: 2) and (orientation: portrait)' },
  { name: 'apple-splash-750x1334.png', width: 750, height: 1334, media: '(device-width: 375px) and (device-height: 667px) and (-webkit-device-pixel-ratio: 2) and (orientation: portrait)' },
  { name: 'apple-splash-828x1792.png', width: 828, height: 1792, media: '(device-width: 414px) and (device-height: 896px) and (-webkit-device-pixel-ratio: 2) and (orientation: portrait)' },
  { name: 'apple-splash-1125x2436.png', width: 1125, height: 2436, media: '(device-width: 375px) and (device-height: 812px) and (-webkit-device-pixel-ratio: 3) and (orientation: portrait)' },
  { name: 'apple-splash-1170x2532.png', width: 1170, height: 2532, media: '(device-width: 390px) and (device-height: 844px) and (-webkit-device-pixel-ratio: 3) and (orientation: portrait)' },
  { name: 'apple-splash-1284x2778.png', width: 1284, height: 2778, media: '(device-width: 428px) and (device-height: 926px) and (-webkit-device-pixel-ratio: 3) and (orientation: portrait)' },
  { name: 'apple-splash-2048x2732.png', width: 2048, height: 2732, media: '(device-width: 1024px) and (device-height: 1366px) and (-webkit-device-pixel-ratio: 2) and (orientation: portrait)' },
  { name: 'apple-splash-2732x2048.png', width: 2732, height: 2048, media: '(device-width: 1024px) and (device-height: 1366px) and (-webkit-device-pixel-ratio: 2) and (orientation: landscape)' }
]

const screenshotSpecs = [
  { name: 'screenshot-desktop', width: 1280, height: 720 },
  { name: 'screenshot-mobile', width: 720, height: 1280 }
]

const baseColor = '#0b1522'

console.log('Generating PWA icons and screenshots...')

const renderSvg = async (svgBuffer, outPath, size, options = {}) => {
  const { background = baseColor, fit = 'contain' } = options
  await sharp(svgBuffer)
    .resize(size, size, { fit, background })
    .png()
    .toFile(outPath)
}

await Promise.all(iconSizes.map(async (size) => {
  const filename = `icon-${size}x${size}.png`
  await renderSvg(svgSource, path.join(iconsDir, filename), size)
}))

await Promise.all(appleSizes.map(async (size) => {
  const filename = `apple-touch-icon-${size}x${size}.png`
  await renderSvg(svgSource, path.join(iconsDir, filename), size)
}))

await Promise.all(maskableSizes.map(async (size) => {
  const paddedSize = Math.round(size * 0.8)
  const pngBuffer = await sharp(svgSource)
    .resize(paddedSize, paddedSize, { fit: 'contain', background: { r: 0, g: 0, b: 0, alpha: 0 } })
    .png()
    .toBuffer()

  await sharp({ create: { width: size, height: size, channels: 4, background: baseColor } })
    .composite([{ input: pngBuffer, gravity: 'center' }])
    .png()
    .toFile(path.join(iconsDir, `icon-${size}x${size}-maskable.png`))
}))

await Promise.all(msSpecs.map(async (spec) => {
  await sharp(svgSource)
    .resize(spec.width, spec.height, { fit: 'contain', background: baseColor })
    .png()
    .toFile(path.join(iconsDir, spec.name))
}))

await Promise.all(monoSizes.map(async (size) => {
  const filename = `icon-monochrome-${size}x${size}.png`
  await renderSvg(svgMonoSource, path.join(iconsDir, filename), size)
}))

await sharp(svgSource)
  .resize(16, 16, { fit: 'contain', background: baseColor })
  .toFile(path.join(iconsDir, 'favicon-16x16.png'))
await sharp(svgSource)
  .resize(32, 32, { fit: 'contain', background: baseColor })
  .toFile(path.join(iconsDir, 'favicon-32x32.png'))
await sharp(svgSource)
  .resize(48, 48, { fit: 'contain', background: baseColor })
  .toFile(path.join(iconsDir, 'favicon-48x48.png'))
await sharp(svgSource)
  .resize(96, 96, { fit: 'contain', background: baseColor })
  .toFile(path.join(iconsDir, 'favicon-96x96.png'))

await sharp([{
  input: await sharp(svgSource).resize(16, 16, { fit: 'contain', background: baseColor }).png().toBuffer(),
  raw: { width: 16, height: 16, channels: 4 }
}, {
  input: await sharp(svgSource).resize(32, 32, { fit: 'contain', background: baseColor }).png().toBuffer(),
  raw: { width: 32, height: 32, channels: 4 }
}, {
  input: await sharp(svgSource).resize(48, 48, { fit: 'contain', background: baseColor }).png().toBuffer(),
  raw: { width: 48, height: 48, channels: 4 }
}])
  .toFile(path.join(publicDir, 'favicon.ico'))

await Promise.all(splashSpecs.map(async (spec) => {
  const svg = `<?xml version="1.0" encoding="UTF-8"?>
<svg width="${spec.width}" height="${spec.height}" viewBox="0 0 ${spec.width} ${spec.height}" xmlns="http://www.w3.org/2000/svg">
  <defs>
    <linearGradient id="b" x1="0" y1="0" x2="1" y2="1">
      <stop offset="0%" stop-color="#0b1522" />
      <stop offset="100%" stop-color="#121f36" />
    </linearGradient>
  </defs>
  <rect width="100%" height="100%" fill="url(#b)" />
  <rect x="48" y="48" width="${spec.width - 96}" height="${spec.height - 96}" rx="28" fill="#08101f" opacity="0.95" />
  <text x="50%" y="45%" text-anchor="middle" font-family="Segoe UI,Roboto,Arial,sans-serif" font-size="${Math.round(spec.height * 0.08)}" fill="#ebf4ff" font-weight="700">SentinelCore</text>
  <text x="50%" y="58%" text-anchor="middle" font-family="Segoe UI,Roboto,Arial,sans-serif" font-size="${Math.round(spec.height * 0.05)}" fill="#8aa4c8">Self-hosted security dashboard</text>
</svg>`
  await sharp(Buffer.from(svg)).png().toFile(path.join(iconsDir, spec.name))
}))

await Promise.all(screenshotSpecs.map(async (spec) => {
  const svg = `<?xml version="1.0" encoding="UTF-8"?>
<svg width="${spec.width}" height="${spec.height}" viewBox="0 0 ${spec.width} ${spec.height}" xmlns="http://www.w3.org/2000/svg">
  <defs>
    <linearGradient id="b" x1="0" y1="0" x2="1" y2="1">
      <stop offset="0%" stop-color="#0b1522" />
      <stop offset="100%" stop-color="#121f36" />
    </linearGradient>
  </defs>
  <rect width="100%" height="100%" fill="url(#b)" />
  <rect x="48" y="48" width="${spec.width - 96}" height="${spec.height - 96}" rx="28" fill="#08101f" opacity="0.95" />
  <text x="50%" y="35%" text-anchor="middle" font-family="Segoe UI,Roboto,Arial,sans-serif" font-size="${Math.round(spec.height * 0.08)}" fill="#ebf4ff" font-weight="700">SentinelCore</text>
  <text x="50%" y="50%" text-anchor="middle" font-family="Segoe UI,Roboto,Arial,sans-serif" font-size="${Math.round(spec.height * 0.05)}" fill="#8aa4c8">Self-hosted security dashboard</text>
  <text x="50%" y="65%" text-anchor="middle" font-family="Segoe UI,Roboto,Arial,sans-serif" font-size="${Math.round(spec.height * 0.04)}" fill="#8aa4c8">Fast, secure, and installable</text>
</svg>`
  await sharp(Buffer.from(svg)).png().toFile(path.join(iconsDir, `${spec.name}.png`))
}))

console.log('PWA assets generated in public/icons')
