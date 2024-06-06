/* eslint-disable dot-notation, no-param-reassign */

class Telepath {
	constructors: {[key: string]: any};

	constructor() {
	  	this.constructors = {};
	}
  
	register(name: any, constructor: any) {
	  	this.constructors[name] = constructor;
	}
  
	unpack(objData: any) {
	  	const packedValuesById: {[key: number]: any} = {};
	  	this.scanForIds(objData, packedValuesById);
	  	const valuesById = {};
	  	return this.unpackWithRefs(objData, packedValuesById, valuesById);
	}
  
	scanForIds(objData: any, packedValuesById: {[key: number]: any}) {
	  	/* descend into objData, indexing any objects with an _id in packedValuesById */
		
	  	if (objData === null || typeof(objData) !== 'object') {
			/* primitive value - nothing to scan */
			return;
	  	}
	  
	  	if (Array.isArray(objData)) {
			/* scan recursively */
			objData.forEach(item => this.scanForIds(item, packedValuesById));
			return;
	  	}
	  
	  	/* objData is an object / dict - check for reserved key names */
	  	let hasReservedKeyNames = false;
	  
	  	if ('_id' in objData) {
			hasReservedKeyNames = true;
			/* index object in packedValuesById */
			packedValuesById[objData['_id']] = objData;
	  	}
	  
	  	if ('_type' in objData || '_val' in objData || '_ref' in objData) {
			hasReservedKeyNames = true;
	  	}
	  
	  	if ('_list' in objData) {
			hasReservedKeyNames = true;
			/* scan list items recursively */
			objData['_list'].forEach(function (item: any) {
				this.scanForIds(item, packedValuesById)
			}.bind(this));
	  	}
	  
	  	if ('_args' in objData) {
			hasReservedKeyNames = true;
			/* scan arguments recursively */
			objData['_args'].forEach(function (item: any) {
				this.scanForIds(item, packedValuesById)
			}.bind(this));
	  	}
	  
	  	if ('_dict' in objData) {
			hasReservedKeyNames = true;
			/* scan dict items recursively */
			// eslint-disable-next-line no-unused-vars
			for (const [key, val] of Object.entries(objData['_dict'])) {
			  this.scanForIds(val, packedValuesById);
			}
	  	}
	  
	  	if (!hasReservedKeyNames) {
			/* scan as a plain dict */
			// eslint-disable-next-line no-unused-vars
			for (const [key, val] of Object.entries(objData)) {
			  this.scanForIds(val, packedValuesById);
			}
	  	}
	}
  
	unpackWithRefs(objData: any, packedValuesById: {[key: number]: any}, valuesById: {[key: number]: any}): any {
	  if (objData === null || typeof(objData) !== 'object') {
		/* primitive value - return unchanged */
		return objData;
	  }
  
	  if (Array.isArray(objData)) {
		/* unpack recursively */
		return objData.map(item => this.unpackWithRefs(item, packedValuesById, valuesById));
	  }
  
	  /* objData is an object / dict - check for reserved key names */
	  let result: {[key: string]: any} | any;
  
	  if ('_ref' in objData) {
		if (objData['_ref'] in valuesById) {
		  /* use previously unpacked instance */
		  result = valuesById[objData['_ref']];
		} else {
		  /* look up packed object and unpack it; this will populate valuesById as a side effect */
		  result = this.unpackWithRefs(
			packedValuesById[objData['_ref']], packedValuesById, valuesById
		  );
		}
	  } else if ('_val' in objData) {
		result = objData['_val'];
	  } else if ('_list' in objData) {
		result = objData['_list'].map(function(item: any) {
			return this.unpackWithRefs(item, packedValuesById, valuesById)
		}.bind(this));
	  } else if ('_dict' in objData) {
		result = {};
		for (const [key, val] of Object.entries(objData['_dict'])) {
		  result[key] = this.unpackWithRefs(val, packedValuesById, valuesById);
		}
	  } else if ('_type' in objData) {
		/* handle as a custom type */
		const constructorId = objData['_type'];
		if (!(constructorId in this.constructors)) {
		  throw new Error('telepath unpack found unknown constructor id: ' + constructorId);
		}
		const constructor = this.constructors[constructorId];
		/* unpack arguments recursively */
		const args = objData['_args'].map(function(arg: any) {
			return this.unpackWithRefs(arg, packedValuesById, valuesById)
		}.bind(this));
		result = new constructor(...args);
	  } else if ('_id' in objData) {
		throw new Error('telepath encountered object with _id but no type specified');
	  } else {
		/* no reserved key names found, so unpack objData as a plain dict and return */
		result = {};
		for (const [key, val] of Object.entries(objData)) {
		  result[key] = this.unpackWithRefs(val, packedValuesById, valuesById);
		}
		return result;
	  }
  
	  if ('_id' in objData) {
		valuesById[objData['_id']] = result;
	  }
  
	  return result;
	}
}
  
  
export {
	Telepath,
};