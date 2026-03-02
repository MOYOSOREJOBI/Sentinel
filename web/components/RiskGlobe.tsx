"use client";

import { useEffect, useRef, useState } from 'react'
import { useI18n } from '../lib/i18n'

export type CountryAgg = { iso2: string; name: string; risk: number; highRisk: number; p95: number }

type CountryMeta = {
  iso2: string
  name: string
  lat: [number, number]
  lon: [number, number]
}

// This remains an approximate country set, but hover/click selection now uses
// actual SVG path hit-testing on the globe UV projection instead of rough
// marker intersections.
const COUNTRY_META: CountryMeta[] = [
  { iso2: 'US', name: 'United States', lat: [24, 50], lon: [-125, -66] },
  { iso2: 'CA', name: 'Canada', lat: [42, 84], lon: [-141, -52] },
  { iso2: 'MX', name: 'Mexico', lat: [14, 33], lon: [-118, -86] },
  { iso2: 'BR', name: 'Brazil', lat: [-34, 5], lon: [-74, -34] },
  { iso2: 'AR', name: 'Argentina', lat: [-56, -21], lon: [-74, -53] },
  { iso2: 'GB', name: 'United Kingdom', lat: [49, 61], lon: [-8, 2] },
  { iso2: 'FR', name: 'France', lat: [42, 51], lon: [-5, 9] },
  { iso2: 'DE', name: 'Germany', lat: [47, 55], lon: [5, 15] },
  { iso2: 'ES', name: 'Spain', lat: [35, 44], lon: [-9, 4] },
  { iso2: 'IT', name: 'Italy', lat: [36, 47], lon: [6, 19] },
  { iso2: 'NL', name: 'Netherlands', lat: [50, 54], lon: [3, 8] },
  { iso2: 'SE', name: 'Sweden', lat: [55, 69], lon: [11, 25] },
  { iso2: 'NO', name: 'Norway', lat: [57, 71], lon: [4, 31] },
  { iso2: 'CH', name: 'Switzerland', lat: [45, 48], lon: [5, 10] },
  { iso2: 'PL', name: 'Poland', lat: [49, 55], lon: [14, 24] },
  { iso2: 'TR', name: 'Turkey', lat: [36, 42], lon: [26, 45] },
  { iso2: 'RU', name: 'Russia', lat: [41, 82], lon: [27, 180] },
  { iso2: 'CN', name: 'China', lat: [18, 53], lon: [73, 135] },
  { iso2: 'JP', name: 'Japan', lat: [24, 46], lon: [122, 148] },
  { iso2: 'KR', name: 'South Korea', lat: [34, 38], lon: [126, 130] },
  { iso2: 'HK', name: 'Hong Kong', lat: [22, 23], lon: [114, 115] },
  { iso2: 'SG', name: 'Singapore', lat: [1, 2], lon: [103, 104] },
  { iso2: 'IN', name: 'India', lat: [8, 37], lon: [68, 97] },
  { iso2: 'AE', name: 'United Arab Emirates', lat: [22, 26], lon: [51, 56] },
  { iso2: 'SA', name: 'Saudi Arabia', lat: [16, 32], lon: [36, 56] },
  { iso2: 'EG', name: 'Egypt', lat: [22, 32], lon: [24, 37] },
  { iso2: 'NG', name: 'Nigeria', lat: [4, 14], lon: [3, 15] },
  { iso2: 'ZA', name: 'South Africa', lat: [-35, -22], lon: [16, 33] },
  { iso2: 'AU', name: 'Australia', lat: [-44, -10], lon: [113, 154] },
  { iso2: 'ID', name: 'Indonesia', lat: [-11, 6], lon: [95, 141] },
]

const SVG_W = 2000
const SVG_H = 1000

function lonToX(lon: number) {
  return ((lon + 180) / 360) * SVG_W
}

function latToY(lat: number) {
  return ((90 - lat) / 180) * SVG_H
}

function boxPath(meta: CountryMeta) {
  const x1 = lonToX(meta.lon[0])
  const x2 = lonToX(meta.lon[1])
  const y1 = latToY(meta.lat[1])
  const y2 = latToY(meta.lat[0])
  return `M${x1},${y1} L${x2},${y1} L${x2},${y2} L${x1},${y2} Z`
}

function riskColor(risk: number) {
  if (risk >= 0.75) return '#ef4444'
  if (risk >= 0.45) return '#f97316'
  if (risk >= 0.2) return '#eab308'
  if (risk > 0) return '#38bdf8'
  return '#7087a5'
}

function setTexture(loader: any, material: any, uri: string) {
  loader.load(uri, (texture: any) => {
    texture.colorSpace = undefined
    texture.wrapS = texture.wrapT = 1000
    material.map = texture
    material.needsUpdate = true
  })
}

export function RiskGlobe({ data, onSelect }: { data: CountryAgg[]; onSelect: (iso2: string) => void }) {
  const { tr } = useI18n()
  const containerRef = useRef<HTMLDivElement>(null)
  const onSelectRef = useRef(onSelect)
  const [hovered, setHovered] = useState<{ iso2: string; name: string; risk: number; incidents: number } | null>(null)
  const [tipXY, setTipXY] = useState({ x: 0, y: 0 })
  const hasUnknownGeo = data.some((row) => String(row.iso2 || '').toUpperCase() === 'XX')

  onSelectRef.current = onSelect

  useEffect(() => {
    const container = containerRef.current
    if (!container || typeof window === 'undefined') return

    let frameId = 0
    let disposed = false
    let renderer: any
    let mountedCanvas: HTMLCanvasElement | null = null
    let svgMapEl: SVGSVGElement | null = null
    let svgCountryEl: SVGSVGElement | null = null
    const cleanups: Array<() => void> = []

    const riskMap = new Map(data.map((row) => [String(row.iso2 || '').toUpperCase(), row]))
    const visibleCountries = COUNTRY_META.filter((meta) => riskMap.has(meta.iso2))
    const fallbackCountries = visibleCountries.length > 0 ? visibleCountries : COUNTRY_META

    Promise.all([import('three'), import('three/examples/jsm/controls/OrbitControls.js')]).then(([THREE, controlsMod]) => {
      if (disposed) return

      const { OrbitControls } = controlsMod as { OrbitControls: any }
      const NS = 'http://www.w3.org/2000/svg'

      svgMapEl = document.createElementNS(NS, 'svg')
      svgCountryEl = document.createElementNS(NS, 'svg')
      for (const el of [svgMapEl, svgCountryEl]) {
        el.setAttribute('viewBox', `0 0 ${SVG_W} ${SVG_H}`)
        el.setAttribute('width', String(SVG_W))
        el.setAttribute('height', String(SVG_H))
        el.style.position = 'fixed'
        el.style.left = '-99999px'
        el.style.top = '0'
        el.style.visibility = 'hidden'
        el.style.pointerEvents = 'none'
        document.body.appendChild(el)
      }

      const paths = fallbackCountries.map((meta) => {
        const path = document.createElementNS(NS, 'path')
        path.setAttribute('d', boxPath(meta))
        path.setAttribute('data-iso2', meta.iso2)
        path.setAttribute('data-name', meta.name)
        const row = riskMap.get(meta.iso2)
        path.setAttribute('fill', riskColor(Number(row?.risk ?? 0)))
        path.setAttribute('stroke', '#0f172a')
        path.setAttribute('stroke-width', '6')
        svgMapEl!.appendChild(path)
        return path
      })

      const point = svgMapEl.createSVGPoint()
      const boxes = paths.map((path) => path.getBBox())
      const overlayUris = paths.map((path) => {
        svgCountryEl!.innerHTML = ''
        const clone = path.cloneNode(true) as SVGPathElement
        clone.setAttribute('fill', '#00c9a2')
        clone.setAttribute('stroke', '#e5f7ff')
        clone.setAttribute('stroke-width', '10')
        svgCountryEl!.appendChild(clone)
        return `data:image/svg+xml;charset=utf-8,${encodeURIComponent(new XMLSerializer().serializeToString(svgCountryEl!))}`
      })

      renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true })
      renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2))
      mountedCanvas = renderer.domElement
      container.appendChild(mountedCanvas)

      const scene = new THREE.Scene()
      scene.fog = new THREE.Fog('#d6dee8', 0, 3.2)

      const camera = new THREE.PerspectiveCamera(38, 1, 0.1, 20)
      camera.position.set(0, 0.15, 3)

      const controls = new OrbitControls(camera, renderer.domElement)
      controls.enablePan = false
      controls.enableZoom = false
      controls.enableDamping = true
      controls.minPolarAngle = Math.PI * 0.32
      controls.maxPolarAngle = Math.PI * 0.68
      controls.autoRotate = true
      controls.autoRotateSpeed = 0.55

      const globeGroup = new THREE.Group()
      scene.add(globeGroup)

      const sphereGeometry = new THREE.IcosahedronGeometry(1, 20)
      const baseMaterial = new THREE.MeshBasicMaterial({
        transparent: true,
        alphaTest: 0.05,
        side: THREE.DoubleSide,
      })
      const strokeMaterial = new THREE.MeshBasicMaterial({
        transparent: true,
        opacity: 0.9,
        depthTest: false,
      })
      const selectionMaterial = new THREE.MeshBasicMaterial({
        transparent: true,
        opacity: 0.88,
        side: THREE.DoubleSide,
        depthTest: false,
      })

      const globeBaseMesh = new THREE.Mesh(sphereGeometry, baseMaterial)
      const globeStrokeMesh = new THREE.Mesh(sphereGeometry, strokeMaterial)
      const globeSelectionMesh = new THREE.Mesh(new THREE.IcosahedronGeometry(1.003, 20), selectionMaterial)
      globeStrokeMesh.renderOrder = 2
      globeSelectionMesh.renderOrder = 3
      globeGroup.add(globeBaseMesh, globeStrokeMesh, globeSelectionMesh)

      globeGroup.add(new THREE.Mesh(
        new THREE.SphereGeometry(1.02, 32, 32),
        new THREE.MeshBasicMaterial({
          color: 0x4cc9f0,
          transparent: true,
          opacity: 0.04,
          wireframe: true,
        }),
      ))

      scene.add(new THREE.AmbientLight(0x89aee6, 1.15))
      const key = new THREE.DirectionalLight(0xbfe8ff, 1.25)
      key.position.set(3, 2, 4)
      scene.add(key)

      const loader = new THREE.TextureLoader()

      const fillMarkup = new XMLSerializer().serializeToString(svgMapEl)
      setTexture(loader, baseMaterial, `data:image/svg+xml;charset=utf-8,${encodeURIComponent(fillMarkup)}`)

      for (const path of Array.from(svgMapEl.querySelectorAll('path'))) {
        path.setAttribute('fill', 'none')
        path.setAttribute('stroke', 'rgba(9, 18, 34, 0.95)')
        path.setAttribute('stroke-width', '8')
      }
      const strokeMarkup = new XMLSerializer().serializeToString(svgMapEl)
      setTexture(loader, strokeMaterial, `data:image/svg+xml;charset=utf-8,${encodeURIComponent(strokeMarkup)}`)

      let hoveredIdx = -1
      const raycaster = new THREE.Raycaster()
      raycaster.far = 5
      const pointer = new THREE.Vector2(-1, -1)

      function selectIdx(nextIdx: number, clientX?: number, clientY?: number) {
        if (nextIdx < 0 || nextIdx >= fallbackCountries.length) {
          hoveredIdx = -1
          setHovered(null)
          globeSelectionMesh.visible = false
          return
        }
        if (nextIdx === hoveredIdx && clientX === undefined && clientY === undefined) return
        hoveredIdx = nextIdx
        globeSelectionMesh.visible = true
        setTexture(loader, selectionMaterial, overlayUris[nextIdx])
        const meta = fallbackCountries[nextIdx]
        const row = riskMap.get(meta.iso2)
        setHovered({
          iso2: meta.iso2,
          name: meta.name,
          risk: Number(row?.risk ?? 0),
          incidents: Number(row?.highRisk ?? 0),
        })
        if (clientX !== undefined && clientY !== undefined) {
          const rect = renderer.domElement.getBoundingClientRect()
          setTipXY({
            x: clientX - rect.left,
            y: clientY - rect.top,
          })
        }
      }

      function findCountryIdx(uv: { x: number; y: number } | undefined) {
        if (!uv) return -1
        point.x = uv.x * SVG_W
        point.y = (1 - uv.y) * SVG_H

        for (let i = 0; i < paths.length; i++) {
          const box = boxes[i]
          if (
            point.x < box.x ||
            point.x > box.x + box.width ||
            point.y < box.y ||
            point.y > box.y + box.height
          ) {
            continue
          }
          if (typeof paths[i].isPointInFill === 'function' && paths[i].isPointInFill(point)) {
            return i
          }
        }
        return -1
      }

      function updatePointer(clientX: number, clientY: number) {
        const rect = renderer.domElement.getBoundingClientRect()
        pointer.x = ((clientX - rect.left) / rect.width) * 2 - 1
        pointer.y = -((clientY - rect.top) / rect.height) * 2 + 1
      }

      function handlePointerMove(event: PointerEvent) {
        updatePointer(event.clientX, event.clientY)
        raycaster.setFromCamera(pointer, camera)
        const hits = raycaster.intersectObject(globeBaseMesh)
        if (!hits.length) {
          selectIdx(-1)
          return
        }
        selectIdx(findCountryIdx(hits[0].uv), event.clientX, event.clientY)
      }

      function handleClick(event: MouseEvent) {
        updatePointer(event.clientX, event.clientY)
        raycaster.setFromCamera(pointer, camera)
        const hits = raycaster.intersectObject(globeBaseMesh)
        if (!hits.length) return
        const idx = findCountryIdx(hits[0].uv)
        if (idx >= 0) {
          onSelectRef.current(fallbackCountries[idx].iso2)
        }
      }

      function handleLeave() {
        selectIdx(-1)
      }

      const resize = () => {
        const width = Math.max(280, container.clientWidth || 480)
        const height = Math.max(320, Math.min(width, 520))
        renderer.setSize(width, height)
        camera.aspect = width / height
        camera.updateProjectionMatrix()
      }

      const ro = new ResizeObserver(resize)
      ro.observe(container)
      resize()
      globeSelectionMesh.visible = false

      renderer.domElement.addEventListener('pointermove', handlePointerMove)
      renderer.domElement.addEventListener('click', handleClick)
      renderer.domElement.addEventListener('pointerleave', handleLeave)

      const onDragStart = () => {
        pointer.set(-1, -1)
      }
      controls.addEventListener('start', onDragStart)

      const render = () => {
        if (disposed) return
        controls.update()
        renderer.render(scene, camera)
        frameId = window.requestAnimationFrame(render)
      }
      render()

      cleanups.push(
        () => window.cancelAnimationFrame(frameId),
        () => ro.disconnect(),
        () => controls.dispose(),
        () => controls.removeEventListener('start', onDragStart),
        () => renderer.domElement.removeEventListener('pointermove', handlePointerMove),
        () => renderer.domElement.removeEventListener('click', handleClick),
        () => renderer.domElement.removeEventListener('pointerleave', handleLeave),
        () => {
          renderer.dispose()
          if (mountedCanvas && mountedCanvas.parentNode === container) {
            container.removeChild(mountedCanvas)
          }
        },
        () => {
          svgMapEl?.remove()
          svgCountryEl?.remove()
        },
      )
    })

    return () => {
      disposed = true
      cleanups.forEach((fn) => fn())
    }
  }, [data])

  return (
    <div className='card' data-testid='risk-globe' style={{ position: 'relative' }}>
      <h3>{tr('globeTitle', 'Risk globe')}</h3>
      {hasUnknownGeo ? (
        <div className='badge degraded' style={{ display: 'inline-flex', marginBottom: 8 }}>
          {tr('globeIncompleteGeo', 'Geo enrichment incomplete — map uses fallback codes.')}
        </div>
      ) : null}
      <div
        ref={containerRef}
        style={{
          width: '100%',
          minHeight: 320,
          borderRadius: 8,
          overflow: 'hidden',
          background: 'radial-gradient(circle at 50% 35%, rgba(30,58,95,0.9), #081321 70%)',
          cursor: 'pointer',
        }}
      />
      {hovered ? (
        <div
          style={{
            position: 'absolute',
            left: `${Math.min(tipXY.x + 80, 360)}px`,
            top: `${Math.max(tipXY.y + 44, 62)}px`,
            zIndex: 8,
            minWidth: 150,
            borderRadius: 6,
            border: '1px solid rgba(91,196,255,0.35)',
            background: 'rgba(6, 18, 36, 0.95)',
            color: '#d7e6fb',
            padding: '6px 10px',
            fontSize: 12,
            pointerEvents: 'none',
          }}
        >
          <strong>{hovered.name}</strong> ({hovered.iso2})<br />
          Risk: {hovered.risk.toFixed(3)}<br />
          Incidents: {hovered.incidents}<br />
          <span style={{ color: '#5bc4ff', fontSize: 11 }}>{tr('globeTooltipClick', 'Click to filter')}</span>
        </div>
      ) : null}
    </div>
  )
}
