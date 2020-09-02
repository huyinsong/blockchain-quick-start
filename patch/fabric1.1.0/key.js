/*
 Copyright 2016 IBM All Rights Reserved.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

	  http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

'use strict';

var Hash = require('../../hash.js');
var utils = require('../../utils.js');
var jsrsa = require('jsrsasign');
var asn1 = jsrsa.asn1;
var KEYUTIL = jsrsa.KEYUTIL;
var ECDSA = jsrsa.ECDSA;

var logger = utils.getLogger('ecdsa/key.js');

/**
 * This module implements the {@link module:api.Key} interface, for ECDSA.
 * @class ECDSA_KEY
 * @extends module:api.Key
 */
module.exports = class ECDSA_KEY {
	/**
	 * this class represents the private or public key of an ECDSA key pair.
	 *
	 * @param {Object} key This must be the "privKeyObj" or "pubKeyObj" part of the object generated by jsrsasign.KEYUTIL.generateKeypair()
	 */
	constructor(key) {
		if (typeof key === 'undefined' || key === null) {
			throw new Error('The key parameter is required by this key class implementation, whether this instance is for the public key or private key');
		}

		if (!key.type || key.type !== 'EC') {
			throw new Error('This key implementation only supports keys generated by jsrsasign.KEYUTIL. It must have a "type" property of value "EC"');
		}

		// prvKeyHex value can be null for public keys, so need to check typeof here
		if (typeof key.prvKeyHex === 'undefined') {
			throw new Error('This key implementation only supports keys generated by jsrsasign.KEYUTIL. It must have a "prvKeyHex" property');
		}

		// pubKeyHex must have a non-null value
		if (!key.pubKeyHex) {
			console.log('This key implementation only supports keys generated by jsrsasign.KEYUTIL. It must have a "pubKeyHex" property');
		}

		this._key = (typeof key === 'undefined') ? null : key;
	}

	/**
	 * @returns {string} a string representation of the hash from a sequence based on the private key bytes
	 */
	getSKI() {
		var buff;

		var pointToOctet = function(key) {
			var byteLen = (key.ecparams.keylen + 7) >> 3;
			let buff = Buffer.allocUnsafe(1 + 2 * byteLen);
			buff[0] = 4; // uncompressed point (https://www.security-audit.com/files/x9-62-09-20-98.pdf, section 4.3.6)
			var xyhex = key.getPublicKeyXYHex();
			var xBuffer = Buffer.from(xyhex.x, 'hex');
			var yBuffer = Buffer.from(xyhex.y, 'hex');
			logger.debug('ECDSA curve param X: %s', xBuffer.toString('hex'));
			logger.debug('ECDSA curve param Y: %s', yBuffer.toString('hex'));
			xBuffer.copy(buff, 1 + byteLen - xBuffer.length);
			yBuffer.copy(buff, 1 + 2 * byteLen - yBuffer.length);
			return buff;
		};

		if (this._key.isPublic) {
			// referencing implementation of the Marshal() method of https://golang.org/src/crypto/elliptic/elliptic.go
			buff = pointToOctet(this._key);
		} else {
			buff = pointToOctet(this.getPublicKey()._key);
		}

		// always use SHA256 regardless of the key size in effect
		return Hash.sha2_256(buff);
	}

	isSymmetric() {
		return false;
	}

	isPrivate() {
		if (typeof this._key.prvKeyHex !== 'undefined' && this._key.prvKeyHex === null)
			return false;
		else
			return true;
	}

	getPublicKey() {
		if (this._key.isPublic)
			return this;
		else {
			var f = new ECDSA({ curve: this._key.curveName });
			f.setPublicKeyHex(this._key.pubKeyHex);
			f.isPrivate = false;
			f.isPublic = true;
			return new ECDSA_KEY(f);
		}
	}

	/**
	 * Generates a CSR/PKCS#10 certificate signing request for this key
	 * @param {string} subjectDN The X500Name for the certificate request in LDAP(RFC 2253) format
	 * @returns {string} PEM-encoded PKCS#10 certificate signing request
	 * @throws Will throw an error if this is not a private key
	 * @throws Will throw an error if CSR generation fails for any other reason
	 */
	generateCSR(subjectDN) {

		//check to see if this is a private key
		if (!this.isPrivate()){
			throw new Error('A CSR cannot be generated from a public key');
		}

		try {
			var csr = asn1.csr.CSRUtil.newCSRPEM({
				subject: { str: asn1.x509.X500Name.ldapToOneline(subjectDN)},
				sbjpubkey: this.getPublicKey()._key,
				sigalg: 'SHA256withECDSA',
				sbjprvkey: this._key
			});
			return csr;
		} catch (err) {
			throw err;
		}
	}

	toBytes() {
		// this is specific to the private key format generated by
		// npm module 'jsrsasign.KEYUTIL'
		if (this.isPrivate()) {
			return KEYUTIL.getPEM(this._key, 'PKCS8PRV');
		} else {
			return KEYUTIL.getPEM(this._key);
		}
	}

	static isInstance(object) {
		if (typeof object._key === 'undefined') {
			return false;
		}

		let key = object._key;
		return (key.type && key.type === 'EC' &&
			typeof key.prvKeyHex !== 'undefined' && // prvKeyHex value can be null for public keys, so need to check typeof here
			typeof key.pubKeyHex === 'string'); // pubKeyHex must have a non-null value
	}
};