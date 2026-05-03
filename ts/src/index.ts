export const ALPHABET = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
const MAX_PRECISION = 31

const decodeMap = new Map<string, number>(
  [...ALPHABET].map((char, index) => [char, index]),
)

export class LocationCodeError extends Error {
  constructor(message: string) {
    super(message)
    this.name = "LocationCodeError"
  }
}

export class LocationCode {
  private readonly value: string

  constructor(value: string) {
    this.value = value.toUpperCase()
  }

  toString(): string {
    return this.value
  }

  isZero(): boolean {
    return this.value.length === 0
  }

  payload(): string {
    if (this.value.length === 0) {
      return ""
    }

    return this.value.slice(0, -1)
  }

  precision(): number {
    if (this.value.length === 0) {
      throw new LocationCodeError("invalid location code")
    }

    const precision = decodeMap.get(this.value[this.value.length - 1])
    if (precision === undefined) {
      throw new LocationCodeError(
        `invalid location code: unknown precision character ${JSON.stringify(this.value[this.value.length - 1])}`,
      )
    }

    return precision
  }
}

export type LocationID = LocationCode

export interface Bounds {
  minLat: number
  maxLat: number
  minLon: number
  maxLon: number
}

export interface DecodedLocation {
  code: LocationCode
  payload: string
  precision: number
  bounds: Bounds
  centerLat: number
  centerLon: number
}

export function newLocationCode(value: string): LocationCode {
  return new LocationCode(value)
}

export function encode(lat: number, lon: number, precision: number): LocationCode {
  if (lat < -90 || lat > 90) {
    throw new LocationCodeError("latitude must be within [-90, 90]")
  }

  if (lon < -180 || lon > 180) {
    throw new LocationCodeError("longitude must be within [-180, 180]")
  }

  if (!Number.isInteger(precision) || precision < 0 || precision > MAX_PRECISION) {
    throw new LocationCodeError(`precision must be within [0, ${MAX_PRECISION}]`)
  }

  const latNorm = clampUnit((lat + 90.0) / 180.0)
  const lonNorm = clampUnit((lon + 180.0) / 360.0)

  const maxValue = 1n << BigInt(precision)
  const latInt = normalizedToInt(latNorm, maxValue)
  const lonInt = normalizedToInt(lonNorm, maxValue)

  const z = interleave(latInt, lonInt, precision)
  const payloadChars = ceilDiv(precision * 2, 5)
  const payload = leftPad(encodeBase32(z), payloadChars, "0")

  return new LocationCode(payload + ALPHABET[precision])
}

export function decode(code: LocationCode | string): DecodedLocation {
  const locationCode = code instanceof LocationCode ? code : new LocationCode(code)
  const value = locationCode.toString()
  if (value.length === 0) {
    throw new LocationCodeError("invalid location code")
  }

  const precision = locationCode.precision()
  const payload = locationCode.payload()
  const expectedPayloadChars = ceilDiv(precision * 2, 5)
  if (payload.length !== expectedPayloadChars) {
    throw new LocationCodeError(
      `invalid location code: payload length ${payload.length} does not match precision ${precision}`,
    )
  }

  const z = decodeBase32(payload)
  const [latInt, lonInt] = deinterleave(z, precision)
  const maxValue = Number(1n << BigInt(precision))

  const latMinNorm = Number(latInt) / maxValue
  const latMaxNorm = Number(latInt + 1n) / maxValue
  const lonMinNorm = Number(lonInt) / maxValue
  const lonMaxNorm = Number(lonInt + 1n) / maxValue

  const bounds: Bounds = {
    minLat: latMinNorm * 180.0 - 90.0,
    maxLat: latMaxNorm * 180.0 - 90.0,
    minLon: lonMinNorm * 360.0 - 180.0,
    maxLon: lonMaxNorm * 360.0 - 180.0,
  }

  return {
    code: locationCode,
    payload,
    precision,
    bounds,
    centerLat: (bounds.minLat + bounds.maxLat) / 2,
    centerLon: (bounds.minLon + bounds.maxLon) / 2,
  }
}

export function parent(code: LocationCode | string, parentPrecision: number): LocationCode {
  const decoded = decode(code)
  if (!Number.isInteger(parentPrecision) || parentPrecision < 0 || parentPrecision > decoded.precision) {
    throw new LocationCodeError(
      `precision must be within [0, ${MAX_PRECISION}]: parent precision ${parentPrecision} exceeds code precision ${decoded.precision}`,
    )
  }

  return encode(decoded.centerLat, decoded.centerLon, parentPrecision)
}

function clampUnit(value: number): number {
  if (value < 0) {
    return 0
  }

  if (value >= 1) {
    return 1 - Number.EPSILON
  }

  return value
}

function normalizedToInt(value: number, maxValue: bigint): bigint {
  if (maxValue === 1n) {
    return 0n
  }

  const result = BigInt(Math.floor(value * Number(maxValue)))
  if (result >= maxValue) {
    return maxValue - 1n
  }

  return result
}

function interleave(latInt: bigint, lonInt: bigint, bitsPerAxis: number): bigint {
  let z = 0n

  for (let i = bitsPerAxis; i > 0; i -= 1) {
    const bit = BigInt(i - 1)
    const lonBit = (lonInt >> bit) & 1n
    const latBit = (latInt >> bit) & 1n

    z = (z << 1n) | lonBit
    z = (z << 1n) | latBit
  }

  return z
}

function deinterleave(z: bigint, bitsPerAxis: number): [bigint, bigint] {
  let latInt = 0n
  let lonInt = 0n

  for (let i = 0; i < bitsPerAxis; i += 1) {
    const shift = BigInt((bitsPerAxis - 1 - i) * 2)
    const lonBit = (z >> (shift + 1n)) & 1n
    const latBit = (z >> shift) & 1n

    lonInt = (lonInt << 1n) | lonBit
    latInt = (latInt << 1n) | latBit
  }

  return [latInt, lonInt]
}

function encodeBase32(value: bigint): string {
  if (value === 0n) {
    return "0"
  }

  const encoded: string[] = []
  let current = value
  while (current > 0n) {
    encoded.push(ALPHABET[Number(current % 32n)])
    current /= 32n
  }

  encoded.reverse()
  return encoded.join("")
}

function decodeBase32(value: string): bigint {
  if (value === "") {
    return 0n
  }

  let decoded = 0n
  for (const char of value) {
    const index = decodeMap.get(char)
    if (index === undefined) {
      throw new LocationCodeError(`invalid location code: invalid payload character ${JSON.stringify(char)}`)
    }

    decoded = decoded * 32n + BigInt(index)
  }

  return decoded
}

function ceilDiv(a: number, b: number): number {
  if (a === 0) {
    return 0
  }

  return Math.floor((a + b - 1) / b)
}

function leftPad(value: string, length: number, pad: string): string {
  if (value.length >= length) {
    return value
  }

  return pad.repeat(length - value.length) + value
}
