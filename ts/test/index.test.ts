import test from "node:test"
import assert from "node:assert/strict"
import { readFileSync } from "node:fs"
import { resolve } from "node:path"

import { LocationCode, LocationCodeError, decode, encode, newLocationCode, parent } from "../src/index.js"

interface TestVector {
  lat: number
  lon: number
  precision: number
  code: string
}

const vectors = JSON.parse(
  readFileSync(resolve(process.cwd(), "../spec/test-vectors.json"), "utf8"),
) as TestVector[]

test("newLocationCode uppercases values", () => {
  const code = newLocationCode("8f3k9zdq2ma")
  assert.equal(code.toString(), "8F3K9ZDQ2MA")
})

test("encode and decode round trip", () => {
  const code = encode(37.7749, -122.4194, 10)
  const decoded = decode(code)

  assert.equal(decoded.code.toString(), code.toString())
  assert.equal(decoded.precision, 10)
  assert.equal(decoded.payload.length, 4)
  assert.ok(decoded.bounds.minLat <= 37.7749)
  assert.ok(decoded.bounds.maxLat >= 37.7749)
  assert.ok(decoded.bounds.minLon <= -122.4194)
  assert.ok(decoded.bounds.maxLon >= -122.4194)
})

test("encode world center", () => {
  assert.equal(encode(0, 0, 10).toString(), "R000A")
})

test("parent contains child center", () => {
  const code = encode(37.7749, -122.4194, 10)
  const parentCode = parent(code, 8)
  const decodedParent = decode(parentCode)
  const decodedChild = decode(code)

  assert.equal(decodedParent.precision, 8)
  assert.ok(decodedParent.bounds.minLat <= decodedChild.centerLat)
  assert.ok(decodedParent.bounds.maxLat >= decodedChild.centerLat)
  assert.ok(decodedParent.bounds.minLon <= decodedChild.centerLon)
  assert.ok(decodedParent.bounds.maxLon >= decodedChild.centerLon)
})

test("payload and precision accessors", () => {
  const code = new LocationCode("8F3K9ZDQ2MA")
  assert.equal(code.precision(), 10)
  assert.equal(code.payload(), "8F3K9ZDQ2M")
})

test("encode rejects invalid inputs", () => {
  assert.throws(() => encode(91, 0, 10), LocationCodeError)
  assert.throws(() => encode(0, 181, 10), LocationCodeError)
  assert.throws(() => encode(0, 0, 32), LocationCodeError)
})

test("decode rejects invalid codes", () => {
  for (const value of ["", "abc", "R000I", "TOOLONGA"]) {
    assert.throws(() => decode(value), LocationCodeError)
  }
})

test("shared spec vectors", () => {
  for (const vector of vectors) {
    assert.equal(encode(vector.lat, vector.lon, vector.precision).toString(), vector.code)
  }
})
