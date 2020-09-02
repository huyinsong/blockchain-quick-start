'use strict';

let log4js = require('log4js');
let logger = log4js.getLogger('InvokeCC');
logger.level = 'DEBUG';

let helper = require('./helper');
let hfc = require('fabric-client');
let util = require('util');

hfc.setLogger(logger);


let invokeChaincode = async function (chaincodeName, channelName, functionName, args,
                                      orderers, orgName, peers, transient, useDiscoverService) {
  logger.debug('\n\n============ Invoke chaincode on org \'' + orgName + '\' ============\n');
  let error_message = '';
  let tx_id_string = null;
  let response_payload = null;

  try {
    logger.debug("Load privateKey and signedCert");
    // first setup the client for this org
    let client = await helper.getClientForOrg(orgName);

    let channel = client.newChannel(channelName);
    //assign orderers to channel
    await orderers.forEach(function(ordererName){
    helper.getOrderer(ordererName).then(function (orderer){
      channel.addOrderer(orderer);
    });
  });
    
  // assign peers to channel
  await peers.forEach(function(peerName){
    helper.getPeer(peerName).then(function(peer){
      channel.addPeer(peer);
    });
  });

    // get an admin based transactionID.
    // This should be set to false, but if that, you must call function:
    // client.setUserContext({username:'admin', password:'adminpw'}, false);
    // to setup a new User object. And since that, you have to enable your CA
    // with you all the time.
    // For convenience, I just do all of these transactions with admin user.
    let tx_id = client.newTransactionID(true);
    tx_id_string = tx_id.getTransactionID();
    let request = {
      targets: peers,
      args: args,
      chaincodeId: chaincodeName,
      chainId: channelName,
      fcn: functionName,
      txId: tx_id,
      transientMap: transient
    };
    let results = await channel.sendTransactionProposal(request, 600000);

    // the returned object has both the endorsement results
    // and the actual proposal, the proposal will be needed
    // later when we send a transaction to the orderer
    let proposalResponses = results[0];
    let proposal = results[1];

    // lets have a look at the responses to see if they are
    // all good, if good they will also include signatures
    // required to be committed
    let all_good = true;
    for (let i in proposalResponses) {
      let one_good = false;
      if (proposalResponses && proposalResponses[i].response &&
        proposalResponses[i].response.status === 200) {
        one_good = true;
        logger.info('invoke success');
      } else {
        let err_detail = 'invoke failed: ' + proposalResponses[i];
        logger.error(err_detail);
        error_message = error_message + err_detail;
      }
      all_good = all_good && one_good;
    }

    if (all_good) {
      logger.info(
        'Successfully sent Proposal and received ProposalResponse: %s',
        JSON.stringify(proposalResponses[0].response));
      response_payload = helper.bufferToString(proposalResponses[0].response.payload, 'utf-8');

      // wait for the channel-based event hub to tell us
      // that the commit was good or bad on each peer in our organization
      let promises = [];
      let event_hubs = channel.getChannelEventHubsForOrg();
      event_hubs.forEach((eh) => {
        logger.debug('invokeEventPromise - setting up event');
        let invokeEventPromise = new Promise((resolve, reject) => {
          let event_timeout = setTimeout(() => {
            let message = 'REQUEST_TIMEOUT:' + eh.getPeerAddr();
            logger.error(message);
            eh.disconnect();
            return [false, message];
          }, 600000);
          eh.registerTxEvent(tx_id_string, (tx, code, block_num) => {
              logger.info('The chaincode invoke transaction has been committed on peer %s', eh.getPeerAddr());
              logger.info('Transaction %s has status of %s in blocl %s', tx, code, block_num);
              clearTimeout(event_timeout);

              if (code !== 'VALID') {
                error_message = util.format('The invoke chaincode transaction was invalid, code:%s', code);
                logger.error(error_message);
                reject(new Error(error_message));
              } else {
                let message = 'The invoke chaincode transaction was valid.';
                logger.info(message);
                resolve(message);
              }
            }, (err) => {
              clearTimeout(event_timeout);
              logger.error(err);
              reject(err);
            },
            // the default for 'unregister' is true for transaction listeners
            // so no real need to set here, however for 'disconnect'
            // the default is false as most event hubs are long running
            // in this use case we are using it only once
            {unregister: true, disconnect: true}
          );
          eh.connect();
        });
        promises.push(invokeEventPromise);
      });

      let orderer_request = {
        txId: tx_id,
        proposalResponses: proposalResponses,
        proposal: proposal
      };
      let sendPromise = channel.sendTransaction(orderer_request);
      // put the send to the orderer last so that the events get registered and
      // are ready for the orderering and committing
      promises.push(sendPromise);
      let results = await Promise.all(promises);
      logger.debug('------->>> R E S P O N S E : %j', results);
      let response = results.pop(); //  orderer results are last in the results
      if (response.status === 'SUCCESS') {
        logger.info('Successfully sent transaction to the orderer.');
      } else {
        error_message = util.format('Failed to order the transaction. Error code: %s', response.status);
        logger.debug(error_message);
      }

      // now see what each of the event hubs reported
      for (let i in results) {
        let event_hub_result = results[i];
        let event_hub = event_hubs[i];
        logger.debug('Event results for event hub :%s', event_hub.getPeerAddr());
        if (typeof event_hub_result === 'string') {
          logger.debug(event_hub_result);
        } else {
          if (!error_message) error_message = event_hub_result.toString();
          logger.debug(event_hub_result.toString());
        }
      }
    } else {
      if (!error_message) {
        error_message = util.format('Failed to send Proposal and receive all good ProposalResponse');
      }
      logger.debug(error_message);
    }
  } catch (error) {
    error_message = util.format('Failed to invoke due to error: ' + error.stack ? error.stack : error);
    logger.error(error_message);
  }
  logger.info('Successfully invoked the chaincode \'%s\' to the channel \'%s\' for transaction ID: %s',
      chaincodeName, channelName, tx_id_string);
    return ['yes', tx_id_string, response_payload];
/** 
  if (!error_message) {
    logger.info('Successfully invoked the chaincode \'%s\' to the channel \'%s\' for transaction ID: %s',
      chaincodeName, channelName, tx_id_string);
    return ['yes', tx_id_string, response_payload];
  } else {
    let message = util.format('Failed to invoke chaincode. cause: %s', error_message);
    logger.error(message);
    return ['no', message];
  }**/
};

let queryChaincode = async function (chaincodeName, channelName, functionName, args,
                                     orderers, orgName, peers, transient, useDiscoverService) {
  logger.debug('\n\n============ Query chaincode on org \'' + orgName + '\' ============\n');
  try {
    logger.debug("Load privateKey and signedCert");
    // first setup the client for this org
    let client = await helper.getClientForOrg_normUsage(orgName);
    let channel = client.newChannel(channelName);
    //assign orderers to channel
    await orderers.forEach(function(ordererName){
    helper.getOrderer(ordererName).then(function (orderer){
      channel.addOrderer(orderer);
    });
  });
    
  // assign peers to channel
  await peers.forEach(function(peerName){
    helper.getPeer(peerName).then(function(peer){
      channel.addPeer(peer);
    });
  });
    let tx_id = client.newTransactionID(true);
    let request = {
      targets: peers,
      chaincodeId: chaincodeName,
      fcn: functionName,
      args: args,
      txId: tx_id,
      request_timeout: 600000
    };
    logger.debug("Make query");
    let response_payloads = await channel.queryByChaincode(request);
    if (response_payloads) {
      let queryResult = [];
      queryResult[0] = true;
      queryResult[1] = [];
      for (let i in response_payloads) {
        if (response_payloads.hasOwnProperty(i)) {
          queryResult[1].push(response_payloads[i].toString());
          logger.info('Query result from peer [%s]: %s', i, response_payloads[i].toString());
        }
      }
      return queryResult;
    } else {
      logger.error('response_payloads is null');
      return [false, 'response_payloads is null'];
    }
  } catch (error) {
    logger.error('Failed to query due to error: ' + error.stack ? error.stack : error);
    return [false, error.toString()];
  }
};

exports.invokeChaincode = invokeChaincode;
exports.queryChaincode = queryChaincode;
