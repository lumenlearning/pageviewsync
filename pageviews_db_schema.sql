/* pageviewsync Source Code
 * Copyright (C) 2013 Lumen LLC. 
 * 
 * pageviewsync is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 * 
 * pageviewsync is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 * 
 * You should have received a copy of the GNU Affero General Public License
 * along with pageviewsync.  If not, see <http://www.gnu.org/licenses/>.
 */


CREATE DATABASE  IF NOT EXISTS `canvas_pageviews` /*!40100 DEFAULT CHARACTER SET latin1 */;
USE `canvas_pageviews`;
-- MySQL dump 10.13  Distrib 5.5.34, for debian-linux-gnu (x86_64)
--
-- Host: localhost    Database: canvas_pageviews
-- ------------------------------------------------------
-- Server version	5.5.34-0ubuntu0.12.04.1

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

--
-- Table structure for table `pageviews`
--

DROP TABLE IF EXISTS `pageviews`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `pageviews` (
  `pageviews_key` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `account_id` bigint(20) DEFAULT NULL,
  `action` varchar(256) DEFAULT NULL,
  `asset_id` bigint(20) DEFAULT NULL,
  `asset_type` varchar(256) DEFAULT NULL,
  `asset_user_access_id` bigint(20) DEFAULT NULL,
  `context_id` bigint(20) DEFAULT NULL,
  `context_type` varchar(256) DEFAULT NULL,
  `contributed` varchar(256) DEFAULT NULL,
  `controller` varchar(256) DEFAULT NULL,
  `created_at` varchar(256) DEFAULT NULL,
  `created_at_unix` int(10) unsigned DEFAULT NULL,
  `developer_key_id` varchar(256) DEFAULT NULL,
  `http_method` varchar(256) DEFAULT NULL,
  `interaction_seconds` bigint(20) DEFAULT NULL,
  `participated` tinyint(1) DEFAULT NULL,
  `real_user_id` varchar(256) DEFAULT NULL,
  `render_time` float DEFAULT NULL,
  `request_id` varchar(256) DEFAULT NULL,
  `session_id` varchar(256) DEFAULT NULL,
  `summarized` tinyint(1) DEFAULT NULL,
  `updated_at` varchar(256) DEFAULT NULL,
  `updated_at_unix` int(10) unsigned DEFAULT NULL,
  `url` varchar(2048) DEFAULT NULL,
  `user_agent` varchar(2048) DEFAULT NULL,
  `user_id` bigint(20) DEFAULT NULL,
  `user_id_requested` bigint(20) DEFAULT NULL,
  `user_request` tinyint(1) DEFAULT NULL,
  `remote_ip` varchar(45) DEFAULT NULL,
  PRIMARY KEY (`pageviews_key`),
  UNIQUE KEY `request_id_unq` (`request_id`),
  KEY `account_id_idx` (`account_id`),
  KEY `context_idx` (`context_id`,`context_type`),
  KEY `user_id_requested_idx` (`user_id_requested`) USING BTREE,
  KEY `created_at_unix_idx` (`created_at_unix`) USING BTREE,
  KEY `updated_at_unix_idx` (`updated_at_unix`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=634582 DEFAULT CHARSET=latin1 PACK_KEYS=1 ROW_FORMAT=COMPRESSED;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on 2014-02-04 11:53:18
