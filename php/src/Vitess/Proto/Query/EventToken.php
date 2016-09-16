<?php
// DO NOT EDIT! Generated by Protobuf-PHP protoc plugin 1.0
// Source: query.proto

namespace Vitess\Proto\Query {

  class EventToken extends \DrSlump\Protobuf\Message {

    /**  @var int */
    public $timestamp = null;
    
    /**  @var string */
    public $shard = null;
    
    /**  @var string */
    public $position = null;
    

    /** @var \Closure[] */
    protected static $__extensions = array();

    public static function descriptor()
    {
      $descriptor = new \DrSlump\Protobuf\Descriptor(__CLASS__, 'query.EventToken');

      // OPTIONAL INT64 timestamp = 1
      $f = new \DrSlump\Protobuf\Field();
      $f->number    = 1;
      $f->name      = "timestamp";
      $f->type      = \DrSlump\Protobuf::TYPE_INT64;
      $f->rule      = \DrSlump\Protobuf::RULE_OPTIONAL;
      $descriptor->addField($f);

      // OPTIONAL STRING shard = 2
      $f = new \DrSlump\Protobuf\Field();
      $f->number    = 2;
      $f->name      = "shard";
      $f->type      = \DrSlump\Protobuf::TYPE_STRING;
      $f->rule      = \DrSlump\Protobuf::RULE_OPTIONAL;
      $descriptor->addField($f);

      // OPTIONAL STRING position = 3
      $f = new \DrSlump\Protobuf\Field();
      $f->number    = 3;
      $f->name      = "position";
      $f->type      = \DrSlump\Protobuf::TYPE_STRING;
      $f->rule      = \DrSlump\Protobuf::RULE_OPTIONAL;
      $descriptor->addField($f);

      foreach (self::$__extensions as $cb) {
        $descriptor->addField($cb(), true);
      }

      return $descriptor;
    }

    /**
     * Check if <timestamp> has a value
     *
     * @return boolean
     */
    public function hasTimestamp(){
      return $this->_has(1);
    }
    
    /**
     * Clear <timestamp> value
     *
     * @return \Vitess\Proto\Query\EventToken
     */
    public function clearTimestamp(){
      return $this->_clear(1);
    }
    
    /**
     * Get <timestamp> value
     *
     * @return int
     */
    public function getTimestamp(){
      return $this->_get(1);
    }
    
    /**
     * Set <timestamp> value
     *
     * @param int $value
     * @return \Vitess\Proto\Query\EventToken
     */
    public function setTimestamp( $value){
      return $this->_set(1, $value);
    }
    
    /**
     * Check if <shard> has a value
     *
     * @return boolean
     */
    public function hasShard(){
      return $this->_has(2);
    }
    
    /**
     * Clear <shard> value
     *
     * @return \Vitess\Proto\Query\EventToken
     */
    public function clearShard(){
      return $this->_clear(2);
    }
    
    /**
     * Get <shard> value
     *
     * @return string
     */
    public function getShard(){
      return $this->_get(2);
    }
    
    /**
     * Set <shard> value
     *
     * @param string $value
     * @return \Vitess\Proto\Query\EventToken
     */
    public function setShard( $value){
      return $this->_set(2, $value);
    }
    
    /**
     * Check if <position> has a value
     *
     * @return boolean
     */
    public function hasPosition(){
      return $this->_has(3);
    }
    
    /**
     * Clear <position> value
     *
     * @return \Vitess\Proto\Query\EventToken
     */
    public function clearPosition(){
      return $this->_clear(3);
    }
    
    /**
     * Get <position> value
     *
     * @return string
     */
    public function getPosition(){
      return $this->_get(3);
    }
    
    /**
     * Set <position> value
     *
     * @param string $value
     * @return \Vitess\Proto\Query\EventToken
     */
    public function setPosition( $value){
      return $this->_set(3, $value);
    }
  }
}
